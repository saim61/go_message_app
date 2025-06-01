package gateway

import (
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Client struct {
	ID       string
	Username string
	Room     string
	Conn     *websocket.Conn
	Send     chan WireMessage
}

type Hub struct {
	clients    map[string]*Client
	rooms      map[string]map[string]*Client // room -> clientID -> client
	register   chan *Client
	unregister chan *Client
	broadcast  chan WireMessage
	mutex      sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		rooms:      make(map[string]map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan WireMessage),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client.ID] = client
			if h.rooms[client.Room] == nil {
				h.rooms[client.Room] = make(map[string]*Client)
			}
			h.rooms[client.Room][client.ID] = client
			h.mutex.Unlock()

			log.Printf("[gateway] User %s joined room %s", client.Username, client.Room)

			// Send join notification
			joinMsg := WireMessage{
				ID:        uuid.NewString(),
				Room:      client.Room,
				Author:    "System",
				Body:      client.Username + " joined the room",
				CreatedAt: time.Now().UTC(),
			}
			h.broadcastToRoom(client.Room, joinMsg)

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client.ID]; ok {
				delete(h.clients, client.ID)
				if roomClients, exists := h.rooms[client.Room]; exists {
					delete(roomClients, client.ID)
					if len(roomClients) == 0 {
						delete(h.rooms, client.Room)
					}
				}
				close(client.Send)
			}
			h.mutex.Unlock()

			log.Printf("[gateway] User %s left room %s", client.Username, client.Room)

			// Send leave notification
			leaveMsg := WireMessage{
				ID:        uuid.NewString(),
				Room:      client.Room,
				Author:    "System",
				Body:      client.Username + " left the room",
				CreatedAt: time.Now().UTC(),
			}
			h.broadcastToRoom(client.Room, leaveMsg)

		case message := <-h.broadcast:
			h.broadcastToRoom(message.Room, message)
		}
	}
}

func (h *Hub) broadcastToRoom(room string, message WireMessage) {
	h.mutex.RLock()
	roomClients := h.rooms[room]
	h.mutex.RUnlock()

	for _, client := range roomClients {
		select {
		case client.Send <- message:
		default:
			close(client.Send)
			h.mutex.Lock()
			delete(h.clients, client.ID)
			delete(h.rooms[room], client.ID)
			h.mutex.Unlock()
		}
	}
}

func (h *Hub) GetRoomUserCount(room string) int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	if roomClients, exists := h.rooms[room]; exists {
		return len(roomClients)
	}
	return 0
}

func (h *Hub) Broadcast(message WireMessage) {
	h.broadcast <- message
}

func (h *Hub) Register(client *Client) {
	h.register <- client
}

func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}
