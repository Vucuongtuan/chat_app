package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	PlatformPhone   = "phone"   // Điện thoại — chỉ 1 session active duy nhất (primary)
	PlatformTablet  = "tablet"  // Máy tính bảng — secondary
	PlatformPC      = "pc"      // Máy tính để bàn — secondary
	PlatformLaptop  = "laptop"  // Laptop — secondary
	PlatformWeb     = "web"     // Trình duyệt web — secondary
	PlatformUnknown = "unknown" // Fallback
)

const MaxSecondaryDevices = 10

// IsPhonePlatform kiểm tra platform có phải phone không
func IsPhonePlatform(platform string) bool {
	return platform == PlatformPhone
}

type DeviceSession struct {
	ID        uuid.UUID `json:"id" gorm:"type:char(36);primaryKey"`
	AccountId uuid.UUID `json:"account_id" gorm:"type:char(36);not null;index"`

	DeviceID   string `json:"device_id" gorm:"type:varchar(100);not null;index"`
	DeviceName string `json:"device_name" gorm:"type:varchar(100);not null"`
	// Platform xác định loại thiết bị: "phone", "tablet", "pc", "laptop", "web"
	// Phone là primary — chỉ 1 active duy nhất; các loại khác là secondary, giới hạn tổng MaxSecondaryDevices
	Platform  string `json:"platform" gorm:"type:varchar(20);not null;default:'web'"`
	IPAddress string `json:"ip_address" gorm:"type:varchar(50)"`
	UserAgent string `json:"user_agent" gorm:"type:text"`

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

// QRLoginSession lưu token tạm thời để secondary device đăng nhập qua QR code.
// Luồng: secondary tạo token → primary scan & approve → secondary poll và nhận session.
type QRLoginSession struct {
	// Token là chuỗi random 32-byte hex, dùng làm primary key và nội dung QR code
	Token string `json:"token" gorm:"type:char(64);primaryKey"`

	// AccountId được điền sau khi primary phone approve
	AccountId *uuid.UUID `json:"account_id,omitempty" gorm:"type:char(36);index"`

	// Thông tin thiết bị secondary muốn đăng nhập
	DeviceID   string `json:"device_id" gorm:"type:varchar(100);not null"`
	DeviceName string `json:"device_name" gorm:"type:varchar(100);not null"`
	Platform   string `json:"platform" gorm:"type:varchar(20);not null;default:'web'"`
	IPAddress  string `json:"ip_address" gorm:"type:varchar(50)"`
	UserAgent  string `json:"user_agent" gorm:"type:text"`

	// Status: "pending" → "approved" | "rejected" | "expired"
	Status string `json:"status" gorm:"type:varchar(20);not null;default:'pending'"`

	// SessionID được điền sau khi approved, để secondary device lấy token
	SessionID *uuid.UUID `json:"session_id,omitempty" gorm:"type:char(36)"`

	ExpiresAt time.Time `json:"expires_at" gorm:"not null;index"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// SecondaryOTPSession lưu OTP tạm thời để secondary device đăng nhập qua Bluetooth+OTP.
// OTP được hiển thị trên primary phone (qua push/WebSocket), secondary nhập vào để xác thực.
type SecondaryOTPSession struct {
	ID uuid.UUID `json:"id" gorm:"type:char(36);primaryKey"`

	// AccountId: tài khoản muốn đăng nhập (xác định qua phone number khi request)
	AccountId uuid.UUID `json:"account_id" gorm:"type:char(36);not null;index"`

	// OTP Code 6 chữ số
	OTPCode string `json:"otp_code" gorm:"type:varchar(10);not null"`

	// Thông tin thiết bị secondary
	DeviceID   string `json:"device_id" gorm:"type:varchar(100);not null"`
	DeviceName string `json:"device_name" gorm:"type:varchar(100);not null"`
	Platform   string `json:"platform" gorm:"type:varchar(20);not null;default:'web'"`
	IPAddress  string `json:"ip_address" gorm:"type:varchar(50)"`
	UserAgent  string `json:"user_agent" gorm:"type:text"`

	IsUsed    bool      `json:"is_used" gorm:"type:boolean;not null;default:false"`
	ExpiresAt time.Time `json:"expires_at" gorm:"not null;index"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

func (q *SecondaryOTPSession) BeforeCreate(tx *gorm.DB) (err error) {
	if q.ID == uuid.Nil {
		q.ID = uuid.New()
	}
	return nil
}
