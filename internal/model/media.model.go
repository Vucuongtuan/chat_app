package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Media struct {
	ID uuid.UUID `json:"id" gorm:"type:char(36);primaryKey"`

	Url  string `json:"url" gorm:"type:text;not null"`
	Type string `json:"type" gorm:"type:varchar(20);not null"` // "image", "video"

	OwnerType string    `json:"owner_type" gorm:"type:varchar(30);not null;index"` // "user_avatar", "user_cover", "post", "message"
	OwnerId   uuid.UUID `json:"owner_id" gorm:"type:char(36);not null;index"`

	SortOrder int `json:"sort_order" gorm:"type:int;default:0"` // thứ tự ảnh khi post có nhiều ảnh

	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

func (m *Media) BeforeCreate(tx *gorm.DB) (err error) {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}
