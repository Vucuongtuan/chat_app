package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Message struct {
	ID uuid.UUID `json:"id" gorm:"type:char(36);primaryKey"`

	RoomId   uuid.UUID `json:"room_id" gorm:"type:char(36);not null;index"`
	SenderId uuid.UUID `json:"sender_id" gorm:"type:char(36);not null;index"`

	Type    string `json:"type" gorm:"type:varchar(10);not null;default:'text'"` // text, image, video, file, voice, system
	Content string `json:"content" gorm:"type:text"`                             // text hoặc caption

	ReplyToId       *uuid.UUID `json:"reply_to_id,omitempty" gorm:"type:char(36)"`
	ForwardedFromId *uuid.UUID `json:"forwarded_from_id,omitempty" gorm:"type:char(36)"` // trỏ tới message gốc nếu là forward

	IsEdited bool       `json:"is_edited" gorm:"default:false"`
	EditedAt *time.Time `json:"edited_at,omitempty"`

	IsDeleted bool       `json:"is_deleted" gorm:"default:false"` // soft delete riêng, không dùng gorm.Model's DeletedAt
	DeletedAt *time.Time `json:"deleted_at,omitempty"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	Reactions []MessageReaction `json:"reactions,omitempty" gorm:"foreignKey:MessageId"`
	Statuses  []MessageStatus   `json:"-" gorm:"foreignKey:MessageId"` // không trả full list ra JSON, tính toán riêng
}

func (m *Message) BeforeCreate(tx *gorm.DB) (err error) {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}

type MessageStatus struct {
	ID uuid.UUID `json:"id" gorm:"type:char(36);primaryKey"`

	MessageId uuid.UUID `json:"message_id" gorm:"type:char(36);not null;index:idx_msg_user,unique"`
	UserId    uuid.UUID `json:"user_id" gorm:"type:char(36);not null;index:idx_msg_user,unique"` // người NHẬN, không phải người gửi
	Status    string    `json:"status" gorm:"type:varchar(10);not null;default:'sent'"`          // sent, delivered, read

	DeliveredAt *time.Time `json:"delivered_at,omitempty"`
	ReadAt      *time.Time `json:"read_at,omitempty"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (s *MessageStatus) BeforeCreate(tx *gorm.DB) (err error) {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

type MessageReaction struct {
	ID uuid.UUID `json:"id" gorm:"type:char(36);primaryKey"`

	MessageId uuid.UUID `json:"message_id" gorm:"type:char(36);not null;index:idx_msg_user_reaction,unique"`
	UserId    uuid.UUID `json:"user_id" gorm:"type:char(36);not null;index:idx_msg_user_reaction,unique"`
	Emoji     string    `json:"emoji" gorm:"type:varchar(10);not null"` // "❤️", "😂", "👍"...

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (r *MessageReaction) BeforeCreate(tx *gorm.DB) (err error) {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
