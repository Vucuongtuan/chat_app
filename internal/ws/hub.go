package ws

import "github.com/google/uuid"

type Client struct {
	RoomId uuid.UUID
	Send   chan []byte
}

type registerEvent struct{ client *Client }
type unregisterEvent struct{ client *Client }
type BroadcastMessage struct {
	RoomId  uuid.UUID
	Payload []byte
}

type Hub struct {
	rooms  map[uuid.UUID]map[*Client]struct{}
	events chan any
	done   chan struct{}
}

func NewHub() *Hub {
	return &Hub{
		rooms:  make(map[uuid.UUID]map[*Client]struct{}),
		events: make(chan any, 256),
		done:   make(chan struct{}),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case <-h.done:
			return
		case ev := <-h.events:
			switch e := ev.(type) {
			case registerEvent:
				c := e.client
				if h.rooms[c.RoomId] == nil {
					h.rooms[c.RoomId] = make(map[*Client]struct{})
				}
				h.rooms[c.RoomId][c] = struct{}{}

			case unregisterEvent:
				h.remove(e.client)

			case *BroadcastMessage:
				for c := range h.rooms[e.RoomId] {
					select {
					case c.Send <- e.Payload:
					default:
						h.remove(c) // slow client: xóa + close ngay
					}
				}
			}
		}
	}
}

func (h *Hub) remove(c *Client) {
	clients, ok := h.rooms[c.RoomId]
	if !ok {
		return
	}
	if _, exists := clients[c]; !exists {
		return
	}
	delete(clients, c)
	close(c.Send)
	if len(clients) == 0 {
		delete(h.rooms, c.RoomId) // dọn room rỗng
	}
}

func (h *Hub) Stop() { close(h.done) }

func (h *Hub) send(ev any) {
	select {
	case h.events <- ev:
	case <-h.done:
	}
}

func (h *Hub) Register(c *Client)   { h.send(registerEvent{c}) }
func (h *Hub) Unregister(c *Client) { h.send(unregisterEvent{c}) }
func (h *Hub) BroadcastToRoom(roomId uuid.UUID, payload []byte) {
	h.send(&BroadcastMessage{RoomId: roomId, Payload: payload})
}
