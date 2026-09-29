package dto

// Pin Message DTOs
type PinMessageRequest struct {
	MessageID string `json:"message_id" binding:"required"`
}

type PinnedMessageResponse struct {
	ID        string           `json:"id"`
	RoomID    string           `json:"room_id"`
	MessageID string           `json:"message_id"`
	PinnedBy  string           `json:"pinned_by"`
	PinnedAt  string           `json:"pinned_at"`
	Message   *MessageResponse `json:"message,omitempty"`
}

// Poll DTOs
type CreatePollRequest struct {
	Question    string   `json:"question" binding:"required"`
	Options     []string `json:"options" binding:"required,min=2,dive,required"`
	IsMultiple  bool     `json:"is_multiple"`
	IsAnonymous bool     `json:"is_anonymous"`
	ExpiresAt   *string  `json:"expires_at,omitempty"`
}

type VotePollRequest struct {
	OptionIDs []string `json:"option_ids" binding:"required,min=1"`
}

type PollOptionResponse struct {
	ID        string   `json:"id"`
	Text      string   `json:"text"`
	VoteCount int      `json:"vote_count"`
	VoterIDs  []string `json:"voter_ids,omitempty"` // Trống nếu is_anonymous = true
	HasVoted  bool     `json:"has_voted"`           // Current user đã vote phương án này chưa
}

type PollResponse struct {
	ID          string               `json:"id"`
	RoomID      string               `json:"room_id"`
	MessageID   *string              `json:"message_id,omitempty"`
	CreatorID   string               `json:"creator_id"`
	Question    string               `json:"question"`
	IsMultiple  bool                 `json:"is_multiple"`
	IsAnonymous bool                 `json:"is_anonymous"`
	TotalVotes  int                  `json:"total_votes"`
	Options     []PollOptionResponse `json:"options"`
	ExpiresAt   *string              `json:"expires_at,omitempty"`
	CreatedAt   string               `json:"created_at"`
}

// Room Schedule / Event Reminder DTOs
type CreateScheduleRequest struct {
	Title       string  `json:"title" binding:"required"`
	Description string  `json:"description"`
	Category    string  `json:"category"`                        // meeting, class, deadline, exam, general
	EventTime   string  `json:"event_time" binding:"required"`   // RFC3339 format (bắt đầu)
	EndTime     *string `json:"end_time,omitempty"`              // RFC3339 format (kết thúc)
	IsAllDay    bool    `json:"is_all_day"`
	Color       string  `json:"color"`                           // Mã màu (vd: #3B82F6)
	Location    string  `json:"location"`                        // Địa điểm / link họp
	RemindAt    *string `json:"remind_at,omitempty"`             // RFC3339 format
}

type ScheduleResponse struct {
	ID          string  `json:"id"`
	RoomID      string  `json:"room_id"`
	CreatorID   string  `json:"creator_id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Category    string  `json:"category"`
	EventTime   string  `json:"event_time"`
	EndTime     *string `json:"end_time,omitempty"`
	IsAllDay    bool    `json:"is_all_day"`
	Color       string  `json:"color"`
	Location    string  `json:"location"`
	RemindAt    *string `json:"remind_at,omitempty"`
	IsNotified  bool    `json:"is_notified"`
	CreatedAt   string  `json:"created_at"`
}

// Calendar View DTOs (Bảng lịch phòng theo tháng)
type CalendarItemResponse struct {
	ID        string  `json:"id"`
	Source    string  `json:"source"`   // "schedule" | "activity"
	Type      string  `json:"type"`     // meeting, class, task, attendance, etc.
	Title     string  `json:"title"`
	StartTime string  `json:"start_time"`
	EndTime   *string `json:"end_time,omitempty"`
	IsAllDay  bool    `json:"is_all_day"`
	Color     string  `json:"color"`
	Location  string  `json:"location,omitempty"`
	Status    string  `json:"status,omitempty"`
}

type CalendarDayResponse struct {
	Date           string                 `json:"date"`             // "2026-10-01"
	Day            int                    `json:"day"`              // 1
	DayOfWeek      int                    `json:"day_of_week"`      // 1 (Monday) -> 7 (Sunday)
	DayOfWeekName  string                 `json:"day_of_week_name"` // "Thứ Hai", "Thứ Ba"...
	IsToday        bool                   `json:"is_today"`         // Là ngày hôm nay
	IsPast         bool                   `json:"is_past"`          // Đã qua
	TotalEvents    int                    `json:"total_events"`
	Events         []CalendarItemResponse `json:"events"`           // Danh sách sự kiện trong ngày (nếu không có thì mảng rỗng)
}

type RoomCalendarResponse struct {
	RoomID             string                `json:"room_id"`
	Year               int                   `json:"year"`
	Month              int                   `json:"month"`
	Today              string                `json:"today"`
	TotalDays          int                   `json:"total_days"`
	TotalEventsInMonth int                   `json:"total_events_in_month"`
	Days               []CalendarDayResponse `json:"days"` // Đầy đủ các ngày trong tháng (từ ngày 1 đến ngày cuối)
}

// Location and Contact sharing message payloads
type LocationMessagePayload struct {
	Latitude  float64 `json:"latitude" binding:"required"`
	Longitude float64 `json:"longitude" binding:"required"`
	Title     string  `json:"title,omitempty"`
	Address   string  `json:"address,omitempty"`
}

type ContactMessagePayload struct {
	UserID      *string `json:"user_id,omitempty"`
	FullName    string  `json:"full_name" binding:"required"`
	Phone       string  `json:"phone" binding:"required"`
	AvatarUrl   string  `json:"avatar_url,omitempty"`
	Description string  `json:"description,omitempty"`
}
