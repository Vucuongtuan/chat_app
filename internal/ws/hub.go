package ws

import (
	"sync"

	"github.com/google/uuid"
)

type Hub struct {
	rooms      map[uuid.UUID]map[*Client]bool // roomId -> set of clients
	register   chan *Client
	unregister chan *Client
	broadcast  chan *BroadcastMessage
	mu         sync.RWMutex
}

type BroadcastMessage struct {
	RoomId  uuid.UUID
	Payload []byte
}

type Client struct {
	RoomId uuid.UUID
	Send   chan []byte
}

func NewHub() *Hub {
	return &Hub{
		rooms:      make(map[uuid.UUID]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *BroadcastMessage),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.rooms[client.RoomId] == nil {
				h.rooms[client.RoomId] = make(map[*Client]bool)
			}
			h.rooms[client.RoomId][client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.rooms[client.RoomId]; ok {
				delete(clients, client)
				close(client.Send)
			}
			h.mu.Unlock()

		case msg := <-h.broadcast:
			h.mu.RLock()
			for client := range h.rooms[msg.RoomId] {
				select {
				case client.Send <- msg.Payload:
				default:
					close(client.Send)
					delete(h.rooms[msg.RoomId], client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) BroadcastToRoom(roomId uuid.UUID, payload []byte) {
	h.broadcast <- &BroadcastMessage{RoomId: roomId, Payload: payload}
}
