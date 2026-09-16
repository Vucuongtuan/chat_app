package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Room struct {
	ID uuid.UUID `json:"id" gorm:"type:char(36);primaryKey"`

	Type string `json:"type" gorm:"type:varchar(10);not null"` // "direct", "group"

	Name        *string `json:"name,omitempty" gorm:"type:varchar(100)"`
	Description *string `json:"description,omitempty" gorm:"type:text"`
	AvatarUrl   *string `json:"avatar_url,omitempty" gorm:"type:text"`

	CreatedBy uuid.UUID `json:"created_by" gorm:"type:char(36);not null"`

	InviteCode *string `json:"invite_code,omitempty" gorm:"type:varchar(20);uniqueIndex"`

	LastMessageId *uuid.UUID `json:"last_message_id,omitempty" gorm:"type:char(36)"` // denormalize để query danh sách room nhanh, khỏi join

	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`

	Members []RoomMember `json:"members,omitempty" gorm:"foreignKey:RoomId"`
}

func (r *Room) BeforeCreate(tx *gorm.DB) (err error) {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

type RoomMember struct {
	ID uuid.UUID `json:"id" gorm:"type:char(36);primaryKey"`

	RoomId uuid.UUID `json:"room_id" gorm:"type:char(36);not null;index:idx_room_user,unique"`
	UserId uuid.UUID `json:"user_id" gorm:"type:char(36);not null;index:idx_room_user,unique"`

	Role     string  `json:"role" gorm:"type:varchar(20);not null;default:'member'"` // owner, admin, member
	Nickname *string `json:"nickname,omitempty" gorm:"type:varchar(100)"`            // biệt danh riêng trong group

	JoinedAt time.Time `json:"joined_at" gorm:"not null"`

	// Trạng thái riêng theo từng user — không phải setting chung của room
	IsMuted    bool `json:"is_muted" gorm:"default:false"`
	IsArchived bool `json:"is_archived" gorm:"default:false"`
	IsPinned   bool `json:"is_pinned" gorm:"default:false"` // ghim room lên đầu list

	LastReadMessageId *uuid.UUID `json:"last_read_message_id,omitempty" gorm:"type:char(36)"` // đọc đến tin nào rồi
	LastReadAt        *time.Time `json:"last_read_at,omitempty"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (rm *RoomMember) BeforeCreate(tx *gorm.DB) (err error) {
	if rm.ID == uuid.Nil {
		rm.ID = uuid.New()
	}
	return nil
}
