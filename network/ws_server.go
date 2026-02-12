package network

import (
	"errors"
	"fmt"
	"game_actor/session"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for now
	},
}

type WSServer struct {
	addr     string
	handler  func(sess session.Session, msg []byte)
	onConnect func(sess session.Session)
	onClose   func(sess session.Session)
}

func NewWSServer(addr string) *WSServer {
	return &WSServer{
		addr: addr,
	}
}

func (s *WSServer) SetHandler(h func(sess session.Session, msg []byte)) {
	s.handler = h
}

func (s *WSServer) SetOnConnect(h func(sess session.Session)) {
	s.onConnect = h
}

func (s *WSServer) SetOnClose(h func(sess session.Session)) {
	s.onClose = h
}

func (s *WSServer) Start() error {
	http.HandleFunc("/ws", s.handleWS)
	return http.ListenAndServe(s.addr, nil)
}

func (s *WSServer) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("Upgrade error: %v\n", err)
		return
	}

	sess := newWSSession(conn)
	
	if s.onConnect != nil {
		s.onConnect(sess)
	}

	defer func() {
		sess.Close()
		if s.onClose != nil {
			s.onClose(sess)
		}
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}
		if s.handler != nil {
			s.handler(sess, message)
		}
	}
}

type wsSession struct {
	id     string
	uid    int64
	conn     *websocket.Conn
	sendChan chan []byte
	mu       sync.Mutex
	closed   bool
}

func newWSSession(conn *websocket.Conn) *wsSession {
	sess := &wsSession{
		id:       fmt.Sprintf("%d", time.Now().UnixNano()), // Simple ID generation
		conn:     conn,
		sendChan: make(chan []byte, 256), // Buffered channel
	}
	go sess.writePump()
	return sess
}

func (s *wsSession) writePump() {
	defer func() {
		s.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-s.sendChan:
			if !ok {
				// The channel was closed
				s.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := s.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(msg)

			// Add queued messages to the current websocket message
			n := len(s.sendChan)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-s.sendChan)
			}

			if err := w.Close(); err != nil {
				return
			}
		}
	}
}

func (s *wsSession) ID() string {
	return s.id
}

func (s *wsSession) UserID() int64 {
	return s.uid
}

func (s *wsSession) SetUserID(uid int64) {
	s.uid = uid
}

func (s *wsSession) Send(msg []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return errors.New("session closed")
	}
	
	select {
	case s.sendChan <- msg:
		return nil
	default:
		return errors.New("send buffer full")
	}
}

func (s *wsSession) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	close(s.sendChan)
	return s.conn.Close()
}
