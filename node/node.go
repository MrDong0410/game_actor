package node

import (
	"context"
	"encoding/json"
	"fmt"
	"game_actor/discovery"
	"game_actor/network"
	"game_actor/service"
	"game_actor/session"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"
)

type GameNodeConfig struct {
	NodeID        string
	Host          string
	Port          int
	EtcdEndpoints []string
	RedisAddr     string // e.g. "localhost:6379"
	ServiceName   string
	TTL           int64
}

type GameNode struct {
	config      *GameNodeConfig
	roomSvc     *service.RoomService
	wsServer    *network.WSServer
	discovery   discovery.Discovery
	redisClient *redis.Client
}

func NewGameNode(config *GameNodeConfig, roomBuilder service.Builder) (*GameNode, error) {
	// Initialize Redis Client if configured
	var redisClient *redis.Client
	var kickPublisher service.KickPublisher

	if config.RedisAddr != "" {
		redisClient = redis.NewClient(&redis.Options{
			Addr: config.RedisAddr,
		})

		// Define publisher function
		kickPublisher = func(uid int64) {
			ctx := context.Background()
			msg := fmt.Sprintf(`{"uid": %d, "source_node": "%s"}`, uid, config.NodeID)
			if err := redisClient.Publish(ctx, "game:kick", msg).Err(); err != nil {
				log.Printf("Failed to publish kick message: %v", err)
			}
		}
	}

	// Initialize RoomService
	roomSvc := service.NewRoomService(roomBuilder, kickPublisher)

	// Initialize WS Server
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	wsServer := network.NewWSServer(addr)

	// Initialize Etcd Discovery
	var d discovery.Discovery
	if len(config.EtcdEndpoints) > 0 {
		var err error
		d, err = discovery.NewEtcdDiscovery(config.EtcdEndpoints)
		if err != nil {
			return nil, fmt.Errorf("failed to create etcd discovery: %w", err)
		}
	}

	node := &GameNode{
		config:      config,
		roomSvc:     roomSvc,
		wsServer:    wsServer,
		redisClient: redisClient,
		discovery:   d,
	}

	// Setup WS handlers
	wsServer.SetHandler(node.handleWSMessage)
	wsServer.SetOnConnect(node.handleWSConnect)
	wsServer.SetOnClose(node.handleWSClose)

	return node, nil
}

func (n *GameNode) Start() error {
	// 1. Start Redis Subscriber
	if n.redisClient != nil {
		go n.subscribeKickChannel()
		// Also register to Redis for OpenResty discovery
		go n.registerToRedis()
	}

	// 2. Start WS Server in a goroutine
	go func() {
		log.Printf("Starting WS server on %s:%d", n.config.Host, n.config.Port)
		if err := n.wsServer.Start(); err != nil {
			log.Fatalf("WS server failed: %v", err)
		}
	}()

	// 3. Register to Etcd
	if n.discovery != nil {
		// Address for Nginx to proxy to (e.g., 127.0.0.1:8080)
		addr := fmt.Sprintf("%s:%d", n.config.Host, n.config.Port)
		log.Printf("Registering service %s at %s", n.config.ServiceName, addr)

		ctx := context.Background()
		if err := n.discovery.Register(ctx, n.config.ServiceName, addr, n.config.TTL); err != nil {
			return fmt.Errorf("failed to register service: %w", err)
		}
	}

	return nil
}

func (n *GameNode) registerToRedis() {
	if n.redisClient == nil {
		return
	}
	// Register self to Redis for OpenResty Dynamic Routing
	// Key: game:nodes:{node_id} -> Value: host:port
	key := fmt.Sprintf("game:nodes:%s", n.config.NodeID)
	value := fmt.Sprintf("%s:%d", n.config.Host, n.config.Port)
	ttl := time.Duration(n.config.TTL) * time.Second

	ticker := time.NewTicker(ttl / 2)
	defer ticker.Stop()

	// Initial registration
	ctx := context.Background()
	if err := n.redisClient.Set(ctx, key, value, ttl).Err(); err != nil {
		log.Printf("Failed to register node to Redis: %v", err)
	}

	for range ticker.C {
		if err := n.redisClient.Set(ctx, key, value, ttl).Err(); err != nil {
			log.Printf("Failed to refresh node registration in Redis: %v", err)
		}
	}
}

func (n *GameNode) subscribeKickChannel() {
	ctx := context.Background()
	pubsub := n.redisClient.Subscribe(ctx, "game:kick")
	defer pubsub.Close()

	ch := pubsub.Channel()
	for msg := range ch {
		// Parse message
		type KickMsg struct {
			UID        int64  `json:"uid"`
			SourceNode string `json:"source_node"`
		}
		var kick KickMsg
		if err := json.Unmarshal([]byte(msg.Payload), &kick); err != nil {
			log.Printf("Invalid kick message: %v", err)
			continue
		}

		// Don't kick yourself if you are the source
		if kick.SourceNode == n.config.NodeID {
			continue
		}

		log.Printf("Received kick request for UID: %d from %s", kick.UID, kick.SourceNode)
		n.roomSvc.KickUser(kick.UID)
	}
}

func (n *GameNode) Stop() {
	if n.discovery != nil {
		n.discovery.Close()
	}
	if n.redisClient != nil {
		n.redisClient.Close()
	}
}

func (n *GameNode) Wait() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	<-c
	n.Stop()
}

func (n *GameNode) GetRoomService() *service.RoomService {
	return n.roomSvc
}

// WebSocket Handlers

func (n *GameNode) handleWSConnect(sess session.Session) {
	log.Printf("New session connected: %s", sess.ID())
}

func (n *GameNode) handleWSClose(sess session.Session) {
	log.Printf("Session closed: %s (UID: %d)", sess.ID(), sess.UserID())
	// Handle user disconnection logic here if needed
	// Note: RoomService.UserLeaveRoom is usually called by game logic,
	// but we might want to clean up if the connection drops unexpectedly.
}

func (n *GameNode) handleWSMessage(sess session.Session, msg []byte) {
	// Simple JSON protocol
	type Request struct {
		RoomID int64           `json:"room_id"`
		UID    int64           `json:"uid"`
		Action string          `json:"action"` // "enter", "leave", "message"
		Data   json.RawMessage `json:"data"`
	}

	var req Request
	if err := json.Unmarshal(msg, &req); err != nil {
		log.Printf("Invalid message format from %s: %v", sess.ID(), err)
		return
	}

	// Bind session to user if UID is provided (Simplified auth)
	if req.UID > 0 {
		sess.SetUserID(req.UID)
	}

	log.Printf("Action: %s, Room: %d, UID: %d", req.Action, req.RoomID, req.UID)

	switch req.Action {
	case "enter":
		if err := n.roomSvc.UserEnterRoom(req.UID, req.RoomID, sess); err != nil {
			log.Printf("UserEnterRoom error: %v", err)
			sess.Send([]byte(fmt.Sprintf(`{"error": "%s"}`, err.Error())))
		} else {
			sess.Send([]byte(`{"status": "ok", "action": "enter"}`))
		}
	case "leave":
		if err := n.roomSvc.UserLeaveRoom(req.UID, req.RoomID); err != nil {
			log.Printf("UserLeaveRoom error: %v", err)
		} else {
			sess.Send([]byte(`{"status": "ok", "action": "leave"}`))
		}
	case "message":
		// Broadcast to room (default channel)
		if room, ok := n.roomSvc.GetRoom(req.RoomID); ok {
			room.Broadcast(fmt.Sprintf("%d", req.RoomID), req.Data)
		} else {
			sess.Send([]byte(`{"error": "room not found"}`))
		}
	default:
		sess.Send([]byte(`{"error": "unknown action"}`))
	}
}
