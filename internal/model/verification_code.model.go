package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type VerificationCode struct {
	ID        uuid.UUID  `json:"id" gorm:"type:char(36);primaryKey"`
	AccountId *uuid.UUID `json:"account_id,omitempty" gorm:"type:char(36);index"`

	Target    string    `json:"target" gorm:"type:varchar(100);not null;index"` // email or phone
	Code      string    `json:"code" gorm:"type:varchar(20);not null;index"`
	Type      string    `json:"type" gorm:"type:varchar(50);not null;index"` // "2fa_login", "2fa_enable", "forgot_password"
	ExpiresAt time.Time `json:"expires_at" gorm:"not null;index"`
	IsUsed    bool      `json:"is_used" gorm:"type:boolean;not null;default:false"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

func (v *VerificationCode) BeforeCreate(tx *gorm.DB) (err error) {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	return nil
}
