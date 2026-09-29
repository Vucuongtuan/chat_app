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

	// Device events
	// EventDevicePhoneReplaced: phone cũ bị đăng xuất vì phone mới login
	EventDevicePhoneReplaced EventType = "device.phone_replaced"
	// EventDeviceSecondaryOTPRequested: secondary device xin OTP, primary phone cần hiển thị
	EventDeviceSecondaryOTPRequested EventType = "device.secondary_otp_requested"

	// Room events
	EventMessagePinned       EventType = "room.message_pinned"
	EventMessageUnpinned     EventType = "room.message_unpinned"
	EventPollCreated         EventType = "poll.created"
	EventPollVoted           EventType = "poll.voted"
	EventScheduleCreated     EventType = "schedule.created"
	EventScheduleDeleted     EventType = "schedule.deleted"

	// Activity events (task, attendance, checklist, billing)
	EventActivityCreated            EventType = "activity.created"
	EventActivityUpdated            EventType = "activity.updated"
	EventActivityParticipantUpdated EventType = "activity.participant_updated"
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
