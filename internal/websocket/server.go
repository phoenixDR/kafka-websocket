package websocket

import (
	"encoding/json"
	"fmt"
	"github.com/phoenixDR/kafka-websocket/internal/utils"
	"log"
	"net/http"
	"strings"
	"sync"

	"golang.org/x/net/websocket"
)

type Client struct {
	Conn       *websocket.Conn
	Subscribed map[string]bool
}

type server struct {
	clients    map[*Client]bool
	broadcast  chan string
	register   chan *Client
	unregister chan *Client
	mutex      sync.Mutex
	mutexRW    sync.RWMutex
}

var wsServer = server{
	broadcast:  make(chan string),
	register:   make(chan *Client),
	unregister: make(chan *Client),
	clients:    make(map[*Client]bool),
}

type wsMessage struct {
	Action string `json:"action"`
	Topic  string `json:"topic"`
}

func (s *server) run() {
	for {
		select {
		case client := <-s.register:
			s.registerClient(client)
		case client := <-s.unregister:
			s.unregisterClient(client)
		case message := <-s.broadcast:
			s.broadcastMessage(message)
		}
	}
}

func (s *server) registerClient(client *Client) {
	s.mutex.Lock()
	s.clients[client] = true
	s.mutex.Unlock()
}

func (s *server) unregisterClient(client *Client) {
	s.mutexRW.RLock()
	if _, ok := s.clients[client]; ok {
		s.mutexRW.RUnlock()
		s.mutexRW.Lock()
		delete(s.clients, client)
		client.Conn.Close()
		s.mutexRW.Unlock()
	} else {
		s.mutexRW.RUnlock()
	}
}

func (s *server) broadcastMessage(message string) {
	s.mutex.Lock()
	for client := range s.clients {
		for topic := range client.Subscribed {
			if client.Subscribed[topic] {
				s.sendMessageToClient(client, message)
			}
		}
	}
	s.mutex.Unlock()
}

func (s *server) sendMessageToClient(client *Client, message string) {
	err := websocket.Message.Send(client.Conn, message)
	if err != nil {
		log.Printf("WebSocket send error: %v", err)
		client.Conn.Close()
		delete(s.clients, client)
	}
}

func handleWebSocketConnection(ws *websocket.Conn) {
	client := &Client{Conn: ws, Subscribed: make(map[string]bool)}
	cfg := utils.LoadConfig()
	wsServer.mutex.Lock()
	wsServer.register <- client
	wsServer.mutex.Unlock()
	defer func() {
		wsServer.unregister <- client
		ws.Close()
	}()

	for {
		var message string
		err := websocket.Message.Receive(ws, &message)
		if err != nil {
			log.Printf("WebSocket receive error: %v", err)
			break
		}

		var msg wsMessage
		if err := json.Unmarshal([]byte(message), &msg); err != nil {
			log.Printf("JSON unmarshal error: %v", err)
			continue
		}

		action := msg.Action
		if action == "" {
			websocket.Message.Send(client.Conn, fmt.Sprintln("Invalid message: missing 'action' key"))
			continue
		}
		action = strings.ToLower(action)

		topic := msg.Topic
		if topic == "" {
			websocket.Message.Send(client.Conn, fmt.Sprintln("Invalid message: missing 'topic' key"))
			continue
		}
		topic = strings.ToLower(topic)

		switch action {
		case "subscribe":
			if topic != cfg.Kafka.Topic {
				websocket.Message.Send(
					client.Conn, fmt.Sprintf("Invalid subscription request for unknown topic: %s", topic))

				continue
			}
			client.Subscribed[topic] = true
		case "unsubscribe":
			client.Subscribed[topic] = false
		default:
			websocket.Message.Send(client.Conn, fmt.Sprintf("Invalid action: %s", action))
		}
	}
}

func checkOrigin(r *http.Request) bool {
	return true
}

func ServeWs(w http.ResponseWriter, r *http.Request) {
	if !checkOrigin(r) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	websocket.Handler(handleWebSocketConnection).ServeHTTP(w, r)
}

func StartWebSocketServer(addr string) {
	go wsServer.run()
	http.HandleFunc("/ws", ServeWs)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatalf("WebSocket server failed to start: %v", err)
	}
}

func BroadcastToSubscribedClients(message string) {
	wsServer.broadcast <- message
}

func NotifyAndDisconnectClients() {
	wsServer.notifyCloseClientsConnections()
}

func (s *server) notifyCloseClientsConnections() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for client := range s.clients {
		websocket.Message.Send(client.Conn, "Server is shutting down. Please try again later.")
		client.Conn.Close()
		delete(s.clients, client)
	}
}
