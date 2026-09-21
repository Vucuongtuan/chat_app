package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DeviceSession struct {
	ID        uuid.UUID `json:"id" gorm:"type:char(36);primaryKey"`
	AccountId uuid.UUID `json:"account_id" gorm:"type:char(36);not null;index"`

	DeviceID   string `json:"device_id" gorm:"type:varchar(100);not null;index"`
	DeviceName string `json:"device_name" gorm:"type:varchar(100);not null"`
	DeviceType string `json:"device_type" gorm:"type:varchar(50);not null;default:'web'"`
	IPAddress  string `json:"ip_address" gorm:"type:varchar(50)"`
	UserAgent  string `json:"user_agent" gorm:"type:text"`

	IsPrimary bool `json:"is_primary" gorm:"type:boolean;not null;default:false"`
	IsActive  bool `json:"is_active" gorm:"type:boolean;not null;default:true"`

	RefreshToken string         `json:"-" gorm:"type:text"`
	LastActiveAt time.Time      `json:"last_active_at" gorm:"autoUpdateTime"`
	CreatedAt    time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt    gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`

	Account *Account `json:"account,omitempty" gorm:"foreignKey:AccountId;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

func (s *DeviceSession) BeforeCreate(tx *gorm.DB) (err error) {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	if s.LastActiveAt.IsZero() {
		s.LastActiveAt = time.Now()
	}
	return nil
}
