package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID uuid.UUID `json:"id" gorm:"type:char(36);primaryKey"`

	FullName    string `json:"name" gorm:"type:varchar(100);not null"`
	DisplayName string `json:"display_name" gorm:"type:varchar(100);not null"`
	AvatarUrl   string `json:"avatar_url" gorm:"type:text"`
	CoverUrl    string `json:"cover_url" gorm:"type:text"`
	Bio         string `json:"bio" gorm:"type:text"`

	Status      string     `json:"status" gorm:"type:varchar(20);not null;default:'offline'"`
	LastLoginAt *time.Time `json:"last_login_at" gorm:"type:timestamp"`

	// Foreign key to Account Table
	AccountId uuid.UUID `json:"-" gorm:"type:char(36);not null;unique"`

	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`

	Posts []Post `json:"posts,omitempty" gorm:"foreignKey:UserId"`
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}
