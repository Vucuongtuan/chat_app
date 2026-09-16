package model

import (
	"time"

	"gorm.io/gorm"
)

type Room struct {
	gorm.Model

	Type string `json:"type" gorm:"type:varchar(10);not null"` // "direct", "group"

	Name        *string `json:"name,omitempty" gorm:"type:varchar(100)"`
	Description *string `json:"description,omitempty" gorm:"type:text"`
	AvatarUrl   *string `json:"avatar_url,omitempty" gorm:"type:text"`

	CreatedBy uint `json:"created_by" gorm:"type:bigint;not null"`

	InviteCode *string `json:"invite_code,omitempty" gorm:"type:varchar(20);uniqueIndex"`

	LastMessageId *uint `json:"last_message_id,omitempty" gorm:"type:bigint"` // denormalize để query danh sách room nhanh, khỏi join

	Members []RoomMember `json:"members,omitempty" gorm:"foreignKey:RoomId"`
}

type RoomMember struct {
	gorm.Model

	RoomId uint `json:"room_id" gorm:"type:bigint;not null;index:idx_room_user,unique"`
	UserId uint `json:"user_id" gorm:"type:bigint;not null;index:idx_room_user,unique"`

	Role     string  `json:"role" gorm:"type:varchar(20);not null;default:'member'"` // owner, admin, member
	Nickname *string `json:"nickname,omitempty" gorm:"type:varchar(100)"`            // biệt danh riêng trong group

	JoinedAt time.Time `json:"joined_at" gorm:"not null"`

	// Trạng thái riêng theo từng user — không phải setting chung của room
	IsMuted    bool `json:"is_muted" gorm:"default:false"`
	IsArchived bool `json:"is_archived" gorm:"default:false"`
	IsPinned   bool `json:"is_pinned" gorm:"default:false"` // ghim room lên đầu list

	LastReadMessageId *uint      `json:"last_read_message_id,omitempty" gorm:"type:bigint"` // đọc đến tin nào rồi
	LastReadAt        *time.Time `json:"last_read_at,omitempty"`
}
