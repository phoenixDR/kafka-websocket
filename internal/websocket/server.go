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

type WsConnection interface {
	Close() error
	Read(msg []byte) (int, error)
	Write(msg []byte) (int, error)
	SendMessage(message string) error
}

type RealConnWrapper struct {
	*websocket.Conn
}

type Client struct {
	Conn       WsConnection
	Subscribed map[string]bool
}

type WsServerInterface interface {
	Run()
	RegisterClient(client *Client)
	UnregisterClient(client *Client)
	BroadcastMessage(message string)
	SendMessageToClient(client *Client, message string)
}

type Server struct {
	Clients    map[*Client]bool
	Broadcast  chan string
	Register   chan *Client
	Unregister chan *Client
	Mutex      sync.Mutex
	MutexRW    sync.RWMutex
}

var wsServer = Server{
	Broadcast:  make(chan string),
	Register:   make(chan *Client),
	Unregister: make(chan *Client),
	Clients:    make(map[*Client]bool),
}

type wsMessage struct {
	Action string `json:"action"`
	Topic  string `json:"topic"`
}

func (s *Server) Run() {
	for {
		select {
		case client := <-s.Register:
			s.RegisterClient(client)
		case client := <-s.Unregister:
			s.UnregisterClient(client)
		case message := <-s.Broadcast:
			s.BroadcastMessage(message)
		}
	}
}

func (s *Server) RegisterClient(client *Client) {
	if client != nil {
		s.Mutex.Lock()
		s.Clients[client] = true
		s.Mutex.Unlock()
	}
}

func (s *Server) UnregisterClient(client *Client) {
	if client != nil {
		s.MutexRW.RLock()
		if _, ok := s.Clients[client]; ok {
			s.MutexRW.RUnlock()
			s.MutexRW.Lock()
			delete(s.Clients, client)
			client.Conn.Close()
			s.MutexRW.Unlock()
		} else {
			s.MutexRW.RUnlock()
		}
	}
}

func (s *Server) BroadcastMessage(message string) {
	s.Mutex.Lock()
	for client := range s.Clients {
		for topic := range client.Subscribed {
			if client.Subscribed[topic] {
				s.SendMessageToClient(client, message)
			}
		}
	}
	s.Mutex.Unlock()
}

func (rcw *RealConnWrapper) SendMessage(message string) error {
	return websocket.Message.Send(rcw.Conn, message)
}

func (s *Server) SendMessageToClient(client *Client, message string) {
	if client != nil {
		if err := client.Conn.SendMessage(message); err != nil {
			log.Printf("WebSocket send error: %v", err)
			client.Conn.Close()
			delete(s.Clients, client)
		}
	}
}

func handleWebSocketConnection(ws *websocket.Conn) {
	client := &Client{Conn: &RealConnWrapper{Conn: ws}, Subscribed: make(map[string]bool)}
	cfg := utils.LoadConfig()
	wsServer.Mutex.Lock()
	wsServer.Register <- client
	wsServer.Mutex.Unlock()
	defer func() {
		wsServer.Unregister <- client
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
			wsServer.SendMessageToClient(client, fmt.Sprintln("Invalid message: missing 'action' key"))
			continue
		}
		action = strings.ToLower(action)

		topic := msg.Topic
		if topic == "" {
			wsServer.SendMessageToClient(client, fmt.Sprintln("Invalid message: missing 'topic' key"))
			continue
		}
		topic = strings.ToLower(topic)

		switch action {
		case "subscribe":
			if topic != cfg.Kafka.Topic {
				wsServer.SendMessageToClient(client, fmt.Sprintf("Invalid subscription request for unknown topic: %s", topic))

				continue
			}
			client.Subscribed[topic] = true
		case "unsubscribe":
			client.Subscribed[topic] = false
		default:
			wsServer.SendMessageToClient(client, fmt.Sprintf("Invalid action: %s", action))
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
	go wsServer.Run()
	http.HandleFunc("/ws", ServeWs)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatalf("WebSocket server failed to start: %v", err)
	}
}

func BroadcastToSubscribedClients(message string) {
	wsServer.Broadcast <- message
}

func NotifyAndDisconnectClients() {
	wsServer.notifyCloseClientsConnections()
}

func (s *Server) notifyCloseClientsConnections() {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	for client := range s.Clients {
		s.SendMessageToClient(client, "Server is shutting down. Please try again later.")
		client.Conn.Close()
		delete(s.Clients, client)
	}
}
