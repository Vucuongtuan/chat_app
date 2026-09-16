package model

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model

	FullName    string `json:"name" gorm:"type:varchar(100);not null"`
	DisplayName string `json:"display_name" gorm:"type:varchar(100);not null"`
	AvatarUrl   string `json:"avatar_url" gorm:"type:text"`
	CoverUrl    string `json:"cover_url" gorm:"type:text"`
	Bio         string `json:"bio" gorm:"type:text"`

	Status      string     `json:"status" gorm:"type:varchar(20);not null;default:'offline'"`
	LastLoginAt *time.Time `json:"last_login_at" gorm:"type:timestamp"`

	// Foreign key to Account Table
	AccountId uint `json:"-" gorm:"type:bigint;not null;unique"`

	//
	Posts []Post `json:"posts,omitempty" gorm:"foreignKey:UserId"`
}
