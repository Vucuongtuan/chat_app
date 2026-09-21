package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Account struct {
	ID uuid.UUID `json:"id" gorm:"type:char(36);primaryKey"`

	Phone            string  `json:"phone" gorm:"type:varchar(15);not null;unique"`
	Mail             *string `json:"mail,omitempty" gorm:"type:varchar(100);unique"`
	PasswordHash     string  `json:"-" gorm:"type:varchar(255);not null"`
	HashAlgo         string  `json:"-" gorm:"type:varchar(20);not null;default:'bcrypt'"`
	IsVerified       bool    `json:"is_verified" gorm:"type:boolean;not null;default:false"`
	TwoFactorEnabled bool    `json:"two_factor_enabled" gorm:"type:boolean;not null;default:false"`

	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`

	// Foreign key to User Table
	User     User            `json:"user" gorm:"foreignKey:AccountId;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Sessions []DeviceSession `json:"sessions,omitempty" gorm:"foreignKey:AccountId;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

func (a *Account) BeforeCreate(tx *gorm.DB) (err error) {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}
