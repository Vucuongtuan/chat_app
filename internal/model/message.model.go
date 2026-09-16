package model

import (
	"time"

	"gorm.io/gorm"
)

type Message struct {
	gorm.Model

	RoomId   uint `json:"room_id" gorm:"type:bigint;not null;index"`
	SenderId uint `json:"sender_id" gorm:"type:bigint;not null;index"`

	Type    string `json:"type" gorm:"type:varchar(10);not null;default:'text'"` // text, image, video, file, voice, system
	Content string `json:"content" gorm:"type:text"`                             // text hoặc caption

	ReplyToId       *uint `json:"reply_to_id,omitempty" gorm:"type:bigint"`
	ForwardedFromId *uint `json:"forwarded_from_id,omitempty" gorm:"type:bigint"` // trỏ tới message gốc nếu là forward

	IsEdited bool       `json:"is_edited" gorm:"default:false"`
	EditedAt *time.Time `json:"edited_at,omitempty"`

	IsDeleted bool       `json:"is_deleted" gorm:"default:false"` // soft delete riêng, không dùng gorm.Model's DeletedAt
	DeletedAt *time.Time `json:"deleted_at,omitempty"`

	Reactions []MessageReaction `json:"reactions,omitempty" gorm:"foreignKey:MessageId"`
	Statuses  []MessageStatus   `json:"-" gorm:"foreignKey:MessageId"` // không trả full list ra JSON, tính toán riêng
}

type MessageStatus struct {
	gorm.Model

	MessageId uint   `json:"message_id" gorm:"type:bigint;not null;index:idx_msg_user,unique"`
	UserId    uint   `json:"user_id" gorm:"type:bigint;not null;index:idx_msg_user,unique"` // người NHẬN, không phải người gửi
	Status    string `json:"status" gorm:"type:varchar(10);not null;default:'sent'"`        // sent, delivered, read

	DeliveredAt *time.Time `json:"delivered_at,omitempty"`
	ReadAt      *time.Time `json:"read_at,omitempty"`
}

type MessageReaction struct {
	gorm.Model

	MessageId uint   `json:"message_id" gorm:"type:bigint;not null;index:idx_msg_user_reaction,unique"`
	UserId    uint   `json:"user_id" gorm:"type:bigint;not null;index:idx_msg_user_reaction,unique"`
	Emoji     string `json:"emoji" gorm:"type:varchar(10);not null"` // "❤️", "😂", "👍"...
}
