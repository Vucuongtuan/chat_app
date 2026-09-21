package eventbus

import "sync"

type EventType string

const (
	EventMessageSent     EventType = "message.sent"
	EventMessageDeleted  EventType = "message.deleted"
	EventMessageEdited   EventType = "message.edited"
	EventReactionAdded   EventType = "reaction.added"
	EventReactionRemoved EventType = "reaction.removed"
	EventRoomCreated     EventType = "room.created"
	EventRoomMemberAdded EventType = "room.member_added"
	EventRoomMemberLeft  EventType = "room.member_left"
	EventUserOnline      EventType = "user.online"
	EventUserOffline     EventType = "user.offline"
)

type Event struct {
	Type    EventType
	Payload any
}

type Handler func(event Event)

// Bus là in-process async event bus. Subscribe trước khi Publish.
// Mỗi handler chạy trên goroutine riêng, không block caller.
type Bus struct {
	mu       sync.RWMutex
	handlers map[EventType][]Handler
}

func New() *Bus {
	return &Bus{handlers: make(map[EventType][]Handler)}
}

func (b *Bus) Subscribe(eventType EventType, h Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], h)
}

func (b *Bus) Publish(event Event) {
	b.mu.RLock()
	handlers := make([]Handler, len(b.handlers[event.Type]))
	copy(handlers, b.handlers[event.Type])
	b.mu.RUnlock()
	for _, h := range handlers {
		h := h
		go h(event)
	}
}
