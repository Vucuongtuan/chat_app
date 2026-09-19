package i18n

import (
	"context"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
)

type Language string

const (
	LangVI Language = "vi"
	LangEN Language = "en"
)

var DefaultLanguage = LangVI

type contextKey string

const langContextKey contextKey = "app_language"

// Success message keys
const (
	MsgSuccess         = "success"
	MsgCreated         = "created"
	MsgUpdated         = "updated"
	MsgDeleted         = "deleted"
	MsgMediaUploaded   = "media_uploaded"
	MsgMediaDeleted    = "media_deleted"
	MsgLoginSuccess             = "login_success"
	MsgRegisterSuccess          = "register_success"
	MsgProfileUpdated           = "profile_updated"
	MsgOTPSent                  = "otp_sent"
	Msg2FAEnabled               = "two_factor_enabled"
	Msg2FADisabled              = "two_factor_disabled"
	MsgDeviceLoggedOut          = "device_logged_out"
	MsgAllOtherDevicesLoggedOut = "all_other_devices_logged_out"
	MsgPrimaryTransferred       = "primary_transferred"
	MsgPasswordResetSuccess     = "password_reset_success"
)

// Error message keys
const (
	ErrBadRequest         = "bad_request"
	ErrUnauthorized       = "unauthorized"
	ErrForbidden          = "forbidden"
	ErrNotFound           = "not_found"
	ErrInternalServer     = "internal_server_error"
	ErrValidation         = "validation_error"
	ErrInvalidCredentials = "invalid_credentials"
	ErrDuplicateResource  = "duplicate_resource"
	ErrDatabase           = "database_error"
	ErrExternalService    = "external_service_error"

	// Auth & Device & 2FA specific error keys
	ErrOnlyPrimaryDeviceAllowed  = "only_primary_device_allowed"
	ErrInvalidOTP                = "invalid_otp"
	ErrOTPExpired                = "otp_expired"
	ErrTwoFactorRequired         = "two_factor_required"
	ErrTwoFactorAlreadyEnabled   = "two_factor_already_enabled"
	ErrTwoFactorNotEnabled       = "two_factor_not_enabled"
	ErrEmailRequiredFor2FA       = "email_required_for_2fa"
	ErrSessionNotFound           = "session_not_found"
	ErrCannotLogoutCurrentDevice = "cannot_logout_current_device"
	ErrInvalidOrExpiredToken     = "invalid_or_expired_token"
	ErrAccountNotFound           = "account_not_found"

	// User-specific error keys
	ErrUserNotFound  = "user_not_found"
	ErrInvalidUserID = "invalid_user_id"

	// Media-specific error keys
	ErrInvalidFile      = "invalid_file"
	ErrFileRequired     = "file_required"
	ErrFileUnsupported  = "file_unsupported"
	ErrMissingOwnerType = "missing_owner_type"
	ErrInvalidOwnerID   = "invalid_owner_id"
	ErrInvalidID        = "invalid_id"
	ErrDeleteFailed     = "delete_failed"
	ErrServer           = "server_error"
)

// Message dictionary by language
var messages = map[Language]map[string]string{
	LangVI: {
		// Success
		MsgSuccess:                  "Thao tác thành công",
		MsgCreated:                  "Khởi tạo thành công",
		MsgUpdated:                  "Cập nhật thành công",
		MsgDeleted:                  "Đã xóa thành công",
		MsgMediaUploaded:            "Tải lên media thành công",
		MsgMediaDeleted:             "Đã xóa media thành công",
		MsgLoginSuccess:             "Đăng nhập thành công",
		MsgRegisterSuccess:          "Đăng ký tài khoản thành công",
		MsgProfileUpdated:           "Cập nhật thông tin thành công",
		MsgOTPSent:                  "Mã xác thực đã được gửi đến email",
		Msg2FAEnabled:               "Đã kích hoạt xác thực 2 bước thành công",
		Msg2FADisabled:              "Đã tắt xác thực 2 bước thành công",
		MsgDeviceLoggedOut:          "Đã đăng xuất thiết bị thành công",
		MsgAllOtherDevicesLoggedOut: "Đã đăng xuất tất cả các thiết bị khác thành công",
		MsgPrimaryTransferred:       "Đã chuyển quyền thiết bị chính thành công",
		MsgPasswordResetSuccess:     "Đặt lại mật khẩu thành công",

		// Common Errors
		ErrBadRequest:         "Yêu cầu không hợp lệ",
		ErrUnauthorized:       "Chưa được xác thực hoặc phiên đăng nhập hết hạn",
		ErrForbidden:          "Bạn không có quyền truy cập tài nguyên này",
		ErrNotFound:           "Không tìm thấy tài nguyên yêu cầu",
		ErrInternalServer:     "Lỗi máy chủ nội bộ, vui lòng thử lại sau",
		ErrValidation:         "Dữ liệu gửi lên không hợp lệ",
		ErrInvalidCredentials: "Email hoặc mật khẩu không chính xác",
		ErrDuplicateResource:  "Tài nguyên đã tồn tại trong hệ thống",
		ErrDatabase:           "Lỗi thao tác cơ sở dữ liệu",
		ErrExternalService:    "Lỗi kết nối dịch vụ bên thứ ba",

		// Auth & Device & 2FA Errors
		ErrOnlyPrimaryDeviceAllowed:  "Chỉ thiết bị chính mới có quyền thực hiện thao tác này",
		ErrInvalidOTP:                "Mã xác thực không hợp lệ",
		ErrOTPExpired:                "Mã xác thực đã hết hạn",
		ErrTwoFactorRequired:         "Tài khoản yêu cầu xác thực 2 bước qua email",
		ErrTwoFactorAlreadyEnabled:   "Xác thực 2 bước đã được kích hoạt trước đó",
		ErrTwoFactorNotEnabled:       "Xác thực 2 bước chưa được kích hoạt",
		ErrEmailRequiredFor2FA:       "Tài khoản cần có email hợp lệ để bật xác thực 2 bước",
		ErrSessionNotFound:           "Phiên thiết bị không tồn tại hoặc đã hết hạn",
		ErrCannotLogoutCurrentDevice: "Không thể thu hồi thiết bị hiện tại qua chức năng quản lý từ xa",
		ErrInvalidOrExpiredToken:     "Mã xác thực hoặc token không hợp lệ hoặc đã hết hạn",
		ErrAccountNotFound:           "Tài khoản không tồn tại",

		// User Errors
		ErrUserNotFound:  "Không tìm thấy người dùng",
		ErrInvalidUserID: "ID người dùng không hợp lệ",

		// Media Errors
		ErrInvalidFile:      "File không hợp lệ",
		ErrFileRequired:     "File không được để trống",
		ErrFileUnsupported:  "Định dạng file không được hỗ trợ: %s",
		ErrMissingOwnerType: "Thiếu trường owner_type",
		ErrInvalidOwnerID:   "owner_id không hợp lệ",
		ErrInvalidID:        "id không hợp lệ",
		ErrDeleteFailed:     "Xóa dữ liệu thất bại",
		ErrServer:           "Lỗi máy chủ",
	},
	LangEN: {
		// Success
		MsgSuccess:                  "Operation successful",
		MsgCreated:                  "Created successfully",
		MsgUpdated:                  "Updated successfully",
		MsgDeleted:                  "Deleted successfully",
		MsgMediaUploaded:            "Media uploaded successfully",
		MsgMediaDeleted:             "Media deleted successfully",
		MsgLoginSuccess:             "Login successful",
		MsgRegisterSuccess:          "Account registered successfully",
		MsgProfileUpdated:           "Profile updated successfully",
		MsgOTPSent:                  "Verification code sent to email",
		Msg2FAEnabled:               "Two-factor authentication enabled successfully",
		Msg2FADisabled:              "Two-factor authentication disabled successfully",
		MsgDeviceLoggedOut:          "Device logged out successfully",
		MsgAllOtherDevicesLoggedOut: "All other devices logged out successfully",
		MsgPrimaryTransferred:       "Primary device transferred successfully",
		MsgPasswordResetSuccess:     "Password reset successfully",

		// Common Errors
		ErrBadRequest:         "Invalid request",
		ErrUnauthorized:       "Unauthorized or session expired",
		ErrForbidden:          "Access forbidden",
		ErrNotFound:           "Resource not found",
		ErrInternalServer:     "Internal server error, please try again later",
		ErrValidation:         "Validation error",
		ErrInvalidCredentials: "Email or password is incorrect",
		ErrDuplicateResource:  "Resource already exists",
		ErrDatabase:           "Database error",
		ErrExternalService:    "External service error",

		// Auth & Device & 2FA Errors
		ErrOnlyPrimaryDeviceAllowed:  "Only the primary device is allowed to perform this action",
		ErrInvalidOTP:                "Invalid verification code",
		ErrOTPExpired:                "Verification code has expired",
		ErrTwoFactorRequired:         "Two-factor authentication via email required",
		ErrTwoFactorAlreadyEnabled:   "Two-factor authentication is already enabled",
		ErrTwoFactorNotEnabled:       "Two-factor authentication is not enabled",
		ErrEmailRequiredFor2FA:       "An email address is required to enable two-factor authentication",
		ErrSessionNotFound:           "Device session not found or already expired",
		ErrCannotLogoutCurrentDevice: "Cannot logout current device via remote management",
		ErrInvalidOrExpiredToken:     "Invalid or expired token",
		ErrAccountNotFound:           "Account not found",

		// User Errors
		ErrUserNotFound:  "User not found",
		ErrInvalidUserID: "Invalid user ID",

		// Media Errors
		ErrInvalidFile:      "Invalid file",
		ErrFileRequired:     "File cannot be empty",
		ErrFileUnsupported:  "File format is not supported: %s",
		ErrMissingOwnerType: "Missing owner_type",
		ErrInvalidOwnerID:   "Invalid owner_id",
		ErrInvalidID:        "Invalid id",
		ErrDeleteFailed:     "Failed to delete",
		ErrServer:           "Server error",
	},
}

// NormalizeLanguage parses language input to supported Language (vi or en)
func NormalizeLanguage(raw string) Language {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return DefaultLanguage
	}

	// Direct match
	if raw == "vi" || strings.HasPrefix(raw, "vi-") || strings.HasPrefix(raw, "vi_") {
		return LangVI
	}
	if raw == "en" || strings.HasPrefix(raw, "en-") || strings.HasPrefix(raw, "en_") {
		return LangEN
	}

	// Parse Accept-Language header format: "en-US,en;q=0.9,vi;q=0.8"
	parts := strings.Split(raw, ",")
	for _, part := range parts {
		token := strings.TrimSpace(strings.Split(part, ";")[0])
		if strings.HasPrefix(token, "vi") {
			return LangVI
		}
		if strings.HasPrefix(token, "en") {
			return LangEN
		}
	}

	return DefaultLanguage
}

// GetLanguage extracts the language from gin.Context
func GetLanguage(c *gin.Context) Language {
	if c == nil {
		return DefaultLanguage
	}

	// 1. Check gin context if set by middleware
	if val, exists := c.Get(string(langContextKey)); exists {
		if lang, ok := val.(Language); ok {
			return lang
		}
	}

	// 2. Query param: ?lang=vi or ?lang=en
	if langQuery := c.Query("lang"); langQuery != "" {
		return NormalizeLanguage(langQuery)
	}

	// 3. Custom Header: X-Language
	if langHeader := c.GetHeader("X-Language"); langHeader != "" {
		return NormalizeLanguage(langHeader)
	}

	// 4. Standard Accept-Language header
	if acceptLang := c.GetHeader("Accept-Language"); acceptLang != "" {
		return NormalizeLanguage(acceptLang)
	}

	return DefaultLanguage
}

// FromContext extracts the language from standard context.Context
func FromContext(ctx context.Context) Language {
	if ctx == nil {
		return DefaultLanguage
	}
	if val := ctx.Value(langContextKey); val != nil {
		if lang, ok := val.(Language); ok {
			return lang
		}
	}
	return DefaultLanguage
}

// WithContext returns a new context with the specified language
func WithContext(ctx context.Context, lang Language) context.Context {
	return context.WithValue(ctx, langContextKey, lang)
}

// Middleware injects detected language into gin.Context and request.Context
func Middleware(defaultLang ...Language) gin.HandlerFunc {
	fallback := DefaultLanguage
	if len(defaultLang) > 0 {
		fallback = defaultLang[0]
	}

	return func(c *gin.Context) {
		lang := fallback

		if q := c.Query("lang"); q != "" {
			lang = NormalizeLanguage(q)
		} else if h := c.GetHeader("X-Language"); h != "" {
			lang = NormalizeLanguage(h)
		} else if a := c.GetHeader("Accept-Language"); a != "" {
			lang = NormalizeLanguage(a)
		}

		c.Set(string(langContextKey), lang)
		c.Request = c.Request.WithContext(WithContext(c.Request.Context(), lang))
		c.Next()
	}
}

// T translates a message key for a given language.
// If key is not found, it tries DefaultLanguage.
// If still not found, it returns the key itself.
func T(lang Language, key string, args ...any) string {
	langDict, ok := messages[lang]
	if !ok {
		langDict = messages[DefaultLanguage]
	}

	msg, exists := langDict[key]
	if !exists {
		// Fallback to default language if different
		if lang != DefaultLanguage {
			if defaultDict, ok := messages[DefaultLanguage]; ok {
				msg = defaultDict[key]
			}
		}
	}

	if msg == "" {
		// Key not registered, return key as-is
		msg = key
	}

	if len(args) > 0 {
		return fmt.Sprintf(msg, args...)
	}
	return msg
}

// TCtx translates using the language stored in context.Context
func TCtx(ctx context.Context, key string, args ...any) string {
	return T(FromContext(ctx), key, args...)
}

// RegisterMessages allows adding or overriding messages dynamically
func RegisterMessages(lang Language, newMessages map[string]string) {
	if _, ok := messages[lang]; !ok {
		messages[lang] = make(map[string]string)
	}
	for k, v := range newMessages {
		messages[lang][k] = v
	}
}
