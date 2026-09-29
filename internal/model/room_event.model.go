package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PinnedMessage lưu thông tin tin nhắn được ghim trong phòng chat
type PinnedMessage struct {
	ID        uuid.UUID  `json:"id" gorm:"type:char(36);primaryKey"`
	RoomId    uuid.UUID  `json:"room_id" gorm:"type:char(36);not null;index"`
	MessageId uuid.UUID  `json:"message_id" gorm:"type:char(36);not null;index:idx_room_pinned_msg,unique"`
	PinnedBy  uuid.UUID  `json:"pinned_by" gorm:"type:char(36);not null"`
	PinnedAt  time.Time  `json:"pinned_at" gorm:"autoCreateTime"`

	Message *Message `json:"message,omitempty" gorm:"foreignKey:MessageId;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

func (p *PinnedMessage) BeforeCreate(tx *gorm.DB) (err error) {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// Poll - cuộc bình chọn trong phòng chat
type Poll struct {
	ID          uuid.UUID    `json:"id" gorm:"type:char(36);primaryKey"`
	RoomId      uuid.UUID    `json:"room_id" gorm:"type:char(36);not null;index"`
	MessageId   *uuid.UUID   `json:"message_id,omitempty" gorm:"type:char(36);index"` // trỏ tới message chứa poll nếu có
	CreatorId   uuid.UUID    `json:"creator_id" gorm:"type:char(36);not null"`
	Question    string       `json:"question" gorm:"type:varchar(500);not null"`
	IsMultiple  bool         `json:"is_multiple" gorm:"default:false"`  // cho phép chọn nhiều phương án
	IsAnonymous bool         `json:"is_anonymous" gorm:"default:false"` // bình chọn ẩn danh
	ExpiresAt   *time.Time   `json:"expires_at,omitempty"`
	CreatedAt   time.Time    `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time    `json:"updated_at" gorm:"autoUpdateTime"`

	Options []PollOption `json:"options,omitempty" gorm:"foreignKey:PollId;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

func (p *Poll) BeforeCreate(tx *gorm.DB) (err error) {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// PollOption - lựa chọn trong cuộc bình chọn
type PollOption struct {
	ID     uuid.UUID  `json:"id" gorm:"type:char(36);primaryKey"`
	PollId uuid.UUID  `json:"poll_id" gorm:"type:char(36);not null;index"`
	Text   string     `json:"text" gorm:"type:varchar(255);not null"`
	Order  int        `json:"order" gorm:"default:0"`

	Votes []PollVote `json:"votes,omitempty" gorm:"foreignKey:OptionId;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

func (o *PollOption) BeforeCreate(tx *gorm.DB) (err error) {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	return nil
}

// PollVote - lượt vote của thành viên
type PollVote struct {
	ID        uuid.UUID `json:"id" gorm:"type:char(36);primaryKey"`
	PollId    uuid.UUID `json:"poll_id" gorm:"type:char(36);not null;index"`
	OptionId  uuid.UUID `json:"option_id" gorm:"type:char(36);not null;index"`
	UserId    uuid.UUID `json:"user_id" gorm:"type:char(36);not null;index:idx_poll_user_opt,unique"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

func (v *PollVote) BeforeCreate(tx *gorm.DB) (err error) {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	return nil
}

// RoomSchedule - lịch hẹn / thông báo nhắc sự kiện trong phòng chat
type RoomSchedule struct {
	ID          uuid.UUID  `json:"id" gorm:"type:char(36);primaryKey"`
	RoomId      uuid.UUID  `json:"room_id" gorm:"type:char(36);not null;index"`
	CreatorId   uuid.UUID  `json:"creator_id" gorm:"type:char(36);not null"`
	Title       string     `json:"title" gorm:"type:varchar(255);not null"`
	Description string     `json:"description" gorm:"type:text"`
	Category    string     `json:"category" gorm:"type:varchar(50);not null;default:'general'"` // meeting, class, deadline, exam, general
	EventTime   time.Time  `json:"event_time" gorm:"not null;index"`                            // Thời điểm bắt đầu
	EndTime     *time.Time `json:"end_time,omitempty"`                                          // Thời điểm kết thúc
	IsAllDay    bool       `json:"is_all_day" gorm:"default:false"`
	Color       string     `json:"color" gorm:"type:varchar(20);default:'#3B82F6'"` // Màu đánh dấu trên bảng lịch
	Location    string     `json:"location" gorm:"type:varchar(255)"`               // Địa điểm hoặc link phòng họp online
	RemindAt    *time.Time `json:"remind_at,omitempty"`                             // Thời điểm gửi thông báo nhắc trước
	IsNotified  bool       `json:"is_notified" gorm:"default:false"`
	CreatedAt   time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
}

func (s *RoomSchedule) BeforeCreate(tx *gorm.DB) (err error) {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}
