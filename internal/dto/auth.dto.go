package dto

type RegisterRequest struct {
	Phone       string `json:"phone" binding:"required"`
	Mail        string `json:"mail"`
	Password    string `json:"password" binding:"required,min=8"`
	FullName    string `json:"full_name" binding:"required"`
	DisplayName string `json:"display_name" binding:"required"`

	DeviceID   string `json:"device_id"`
	DeviceName string `json:"device_name"`
	// Platform: "phone", "tablet", "pc", "laptop", "web"
	Platform string `json:"platform"`
	// DeviceType is accepted for backwards compatibility; prefer Platform.
	DeviceType string `json:"device_type,omitempty"`
}

type LoginRequest struct {
	Identifier string `json:"identifier" binding:"required"` // phone or email
	Password   string `json:"password" binding:"required"`

	DeviceID   string `json:"device_id"`
	DeviceName string `json:"device_name"`
	Platform   string `json:"platform"`
	// DeviceType is accepted for backwards compatibility; prefer Platform.
	DeviceType string `json:"device_type,omitempty"`
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
	Platform   string `json:"platform"`
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

// QRChallengeRequest được gọi bởi secondary device để tạo QR token
type QRChallengeRequest struct {
	DeviceID   string `json:"device_id"`
	DeviceName string `json:"device_name" binding:"required"`
	Platform   string `json:"platform" binding:"required"`
}

// QRChallengeResponse trả về token để secondary hiển thị thành QR code
type QRChallengeResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
}

// QRStatusResponse được trả khi secondary device poll trạng thái QR
type QRStatusResponse struct {
	Status       string `json:"status"` // "pending", "approved", "rejected", "expired"
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

// QRApproveRequest được gọi bởi primary phone khi approve QR login
type QRApproveRequest struct {
	Token string `json:"token" binding:"required"`
}

// SecondaryOTPRequest được gọi bởi secondary device để xin OTP
// Server sẽ push OTP đến primary phone qua WebSocket/push notification
type SecondaryOTPRequest struct {
	// Phone của tài khoản muốn đăng nhập
	Phone      string `json:"phone" binding:"required"`
	DeviceID   string `json:"device_id"`
	DeviceName string `json:"device_name" binding:"required"`
	Platform   string `json:"platform" binding:"required"`
}

// SecondaryOTPResponse trả về ID phiên OTP để secondary tracking
type SecondaryOTPResponse struct {
	OTPSessionID string `json:"otp_session_id"`
	ExpiresAt    string `json:"expires_at"`
}

// SecondaryOTPVerifyRequest được gọi bởi secondary device để xác thực OTP
type SecondaryOTPVerifyRequest struct {
	OTPSessionID string `json:"otp_session_id" binding:"required"`
	OTPCode      string `json:"otp_code" binding:"required"`
}

type DeviceSessionResponse struct {
	ID           string `json:"id"`
	DeviceID     string `json:"device_id"`
	DeviceName   string `json:"device_name"`
	Platform     string `json:"platform"`
	IPAddress    string `json:"ip_address"`
	IsPrimary    bool   `json:"is_primary"`
	IsActive     bool   `json:"is_active"`
	IsCurrent    bool   `json:"is_current"`
	LastActiveAt string `json:"last_active_at"`
	CreatedAt    string `json:"created_at"`
}
