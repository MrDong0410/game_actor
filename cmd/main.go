package main

import (
	"flag"
	"game_actor/match"
	"game_actor/node"
	"game_actor/room"
	"log"
)

func main() {
	port := flag.Int("port", 8080, "server port")
	nodeID := flag.String("node", "node-1", "node id")
	flag.Parse()

	config := &node.GameNodeConfig{
		NodeID:        *nodeID,
		Host:          "127.0.0.1",
		Port:          *port,
		EtcdEndpoints: []string{}, // Empty for local test
		ServiceName:   "game-service",
		TTL:           10,
	}

	// Room Builder: Create a RoomActor for each room
	builder := func(roomID int64, matchInfo *match.MatchInfo) room.GameRoom {
		return room.NewRoomActor(roomID, matchInfo)
	}

	gameNode := node.NewGameNode(config, builder)

	// Create a demo room for testing
	demoMatchInfo := &match.MatchInfo{
		MaxPlayerWaitTime: 60,
		MaxGameTime:       300,
		Players:           []*match.Player{},
	}
	
	if _, err := gameNode.GetRoomService().CreateRoom(1, demoMatchInfo); err != nil {
		log.Printf("Failed to create demo room: %v", err)
	} else {
		log.Println("Created demo room with ID: 1")
	}

	log.Printf("Starting GameNode %s on port %d...", *nodeID, *port)
	if err := gameNode.Start(); err != nil {
		log.Fatalf("Failed to start node: %v", err)
	}

	gameNode.Wait()
}
