package dto

type RegisterRequest struct {
	Phone       string `json:"phone" binding:"required"`
	Mail        string `json:"mail"`
	Password    string `json:"password" binding:"required,min=8"`
	FullName    string `json:"full_name" binding:"required"`
	DisplayName string `json:"display_name" binding:"required"`

	DeviceID   string `json:"device_id"`
	DeviceName string `json:"device_name"`
	DeviceType string `json:"device_type"`
}

type LoginRequest struct {
	Identifier string `json:"identifier" binding:"required"` // phone or email
	Password   string `json:"password" binding:"required"`

	DeviceID   string `json:"device_id"`
	DeviceName string `json:"device_name"`
	DeviceType string `json:"device_type"`
}

type AuthResponse struct {
	TwoFactorRequired bool                   `json:"two_factor_required"`
	TempToken         string                 `json:"temp_token,omitempty"`
	AccessToken       string                 `json:"access_token,omitempty"`
	RefreshToken      string                 `json:"refresh_token,omitempty"`
	User              *UserResponse          `json:"user,omitempty"`
	Session           *DeviceSessionResponse `json:"session,omitempty"`
}

type Verify2FARequest struct {
	TempToken  string `json:"temp_token" binding:"required"`
	OTPCode    string `json:"otp_code" binding:"required"`
	DeviceID   string `json:"device_id"`
	DeviceName string `json:"device_name"`
	DeviceType string `json:"device_type"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetPasswordRequest struct {
	Email       string `json:"email" binding:"required,email"`
	OTPCode     string `json:"otp_code" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type Enable2FAConfirmRequest struct {
	OTPCode string `json:"otp_code" binding:"required"`
}

type Disable2FARequest struct {
	Password string `json:"password" binding:"required"`
}

type TransferPrimaryRequest struct {
	TargetSessionID string `json:"target_session_id" binding:"required"`
	Password        string `json:"password" binding:"required"`
}

type DeviceSessionResponse struct {
	ID           string `json:"id"`
	DeviceID     string `json:"device_id"`
	DeviceName   string `json:"device_name"`
	DeviceType   string `json:"device_type"`
	IPAddress    string `json:"ip_address"`
	IsPrimary    bool   `json:"is_primary"`
	IsActive     bool   `json:"is_active"`
	IsCurrent    bool   `json:"is_current"`
	LastActiveAt string `json:"last_active_at"`
	CreatedAt    string `json:"created_at"`
}
