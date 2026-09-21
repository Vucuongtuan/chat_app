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
	MsgSuccess                  = "success"
	MsgCreated                  = "created"
	MsgUpdated                  = "updated"
	MsgDeleted                  = "deleted"
	MsgMediaUploaded            = "media_uploaded"
	MsgMediaDeleted             = "media_deleted"
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

	// Room specific success keys
	MsgRoomCreated   = "room_created"
	MsgRoomUpdated   = "room_updated"
	MsgRoomDeleted   = "room_deleted"
	MsgMemberAdded   = "member_added"
	MsgMemberRemoved = "member_removed"
	MsgMemberUpdated = "member_updated"
	MsgRoomJoined    = "room_joined"

	// Message specific success keys
	MsgMessageSent          = "message_sent"
	MsgMessageEdited        = "message_edited"
	MsgMessageDeleted       = "message_deleted"
	MsgReactionAdded        = "reaction_added"
	MsgReactionRemoved      = "reaction_removed"
	MsgMessageStatusUpdated = "message_status_updated"

	// Post specific success keys
	MsgPostCreated    = "post_created"
	MsgPostUpdated    = "post_updated"
	MsgPostDeleted    = "post_deleted"
	MsgPostLiked      = "post_liked"
	MsgPostUnliked    = "post_unliked"
	MsgCommentAdded   = "comment_added"
	MsgCommentDeleted = "comment_deleted"
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

	// Room specific error keys
	ErrRoomNotFound        = "room_not_found"
	ErrNotRoomMember       = "not_room_member"
	ErrRoomForbidden       = "room_forbidden"
	ErrInvalidRoomID       = "invalid_room_id"
	ErrInvalidInviteCode   = "invalid_invite_code"
	ErrMemberAlreadyExists = "member_already_exists"
	ErrMemberNotFound      = "member_not_found"

	// Message specific error keys
	ErrMessageNotFound  = "message_not_found"
	ErrMessageForbidden = "message_forbidden"
	ErrInvalidMessageID = "invalid_message_id"

	// Post specific error keys
	ErrPostNotFound     = "post_not_found"
	ErrPostForbidden    = "post_forbidden"
	ErrInvalidPostID    = "invalid_post_id"
	ErrCommentNotFound  = "comment_not_found"
	ErrInvalidCommentID = "invalid_comment_id"
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

		// Room Success
		MsgRoomCreated:   "Tạo phòng chat thành công",
		MsgRoomUpdated:   "Cập nhật phòng chat thành công",
		MsgRoomDeleted:   "Xóa phòng chat thành công",
		MsgMemberAdded:   "Thêm thành viên thành công",
		MsgMemberRemoved: "Đã xóa thành viên khỏi phòng chat",
		MsgMemberUpdated: "Cập nhật thành viên thành công",
		MsgRoomJoined:    "Tham gia phòng chat thành công",

		// Message Success
		MsgMessageSent:          "Gửi tin nhắn thành công",
		MsgMessageEdited:        "Chỉnh sửa tin nhắn thành công",
		MsgMessageDeleted:       "Xóa tin nhắn thành công",
		MsgReactionAdded:        "Thả cảm xúc thành công",
		MsgReactionRemoved:      "Gỡ cảm xúc thành công",
		MsgMessageStatusUpdated: "Cập nhật trạng thái tin nhắn thành công",

		// Post Success
		MsgPostCreated:    "Đăng bài viết thành công",
		MsgPostUpdated:    "Cập nhật bài viết thành công",
		MsgPostDeleted:    "Xóa bài viết thành công",
		MsgPostLiked:      "Đã thích bài viết",
		MsgPostUnliked:    "Đã bỏ thích bài viết",
		MsgCommentAdded:   "Thêm bình luận thành công",
		MsgCommentDeleted: "Xóa bình luận thành công",

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

		// Room Errors
		ErrRoomNotFound:        "Không tìm thấy phòng chat",
		ErrNotRoomMember:       "Bạn không phải thành viên của phòng chat này",
		ErrRoomForbidden:       "Bạn không có quyền thực hiện thao tác này trong phòng chat",
		ErrInvalidRoomID:       "ID phòng chat không hợp lệ",
		ErrInvalidInviteCode:   "Mã mời không hợp lệ",
		ErrMemberAlreadyExists: "Thành viên đã có trong phòng chat",
		ErrMemberNotFound:      "Không tìm thấy thành viên trong phòng chat",

		// Message Errors
		ErrMessageNotFound:  "Không tìm thấy tin nhắn",
		ErrMessageForbidden: "Bạn không có quyền thao tác trên tin nhắn này",
		ErrInvalidMessageID: "ID tin nhắn không hợp lệ",

		// Post Errors
		ErrPostNotFound:     "Không tìm thấy bài viết",
		ErrPostForbidden:    "Bạn không có quyền thao tác trên bài viết này",
		ErrInvalidPostID:    "ID bài viết không hợp lệ",
		ErrCommentNotFound:  "Không tìm thấy bình luận",
		ErrInvalidCommentID: "ID bình luận không hợp lệ",
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

		// Room Success
		MsgRoomCreated:   "Room created successfully",
		MsgRoomUpdated:   "Room updated successfully",
		MsgRoomDeleted:   "Room deleted successfully",
		MsgMemberAdded:   "Member added successfully",
		MsgMemberRemoved: "Member removed successfully",
		MsgMemberUpdated: "Member updated successfully",
		MsgRoomJoined:    "Joined room successfully",

		// Message Success
		MsgMessageSent:          "Message sent successfully",
		MsgMessageEdited:        "Message edited successfully",
		MsgMessageDeleted:       "Message deleted successfully",
		MsgReactionAdded:        "Reaction added successfully",
		MsgReactionRemoved:      "Reaction removed successfully",
		MsgMessageStatusUpdated: "Message status updated successfully",

		// Post Success
		MsgPostCreated:    "Post created successfully",
		MsgPostUpdated:    "Post updated successfully",
		MsgPostDeleted:    "Post deleted successfully",
		MsgPostLiked:      "Post liked",
		MsgPostUnliked:    "Post unliked",
		MsgCommentAdded:   "Comment added successfully",
		MsgCommentDeleted: "Comment deleted successfully",

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

		// Room Errors
		ErrRoomNotFound:        "Room not found",
		ErrNotRoomMember:       "You are not a member of this room",
		ErrRoomForbidden:       "You do not have permission in this room",
		ErrInvalidRoomID:       "Invalid room ID",
		ErrInvalidInviteCode:   "Invalid invite code",
		ErrMemberAlreadyExists: "Member already exists in this room",
		ErrMemberNotFound:      "Member not found in this room",

		// Message Errors
		ErrMessageNotFound:  "Message not found",
		ErrMessageForbidden: "You do not have permission for this message",
		ErrInvalidMessageID: "Invalid message ID",

		// Post Errors
		ErrPostNotFound:     "Post not found",
		ErrPostForbidden:    "You do not have permission for this post",
		ErrInvalidPostID:    "Invalid post ID",
		ErrCommentNotFound:  "Comment not found",
		ErrInvalidCommentID: "Invalid comment ID",
	},
}

func NormalizeLanguage(raw string) Language {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return DefaultLanguage
	}

	if raw == "vi" || strings.HasPrefix(raw, "vi-") || strings.HasPrefix(raw, "vi_") {
		return LangVI
	}
	if raw == "en" || strings.HasPrefix(raw, "en-") || strings.HasPrefix(raw, "en_") {
		return LangEN
	}

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

func GetLanguage(c *gin.Context) Language {
	if c == nil {
		return DefaultLanguage
	}

	if val, exists := c.Get(string(langContextKey)); exists {
		if lang, ok := val.(Language); ok {
			return lang
		}
	}

	if langQuery := c.Query("lang"); langQuery != "" {
		return NormalizeLanguage(langQuery)
	}

	if langHeader := c.GetHeader("X-Language"); langHeader != "" {
		return NormalizeLanguage(langHeader)
	}

	if acceptLang := c.GetHeader("Accept-Language"); acceptLang != "" {
		return NormalizeLanguage(acceptLang)
	}

	return DefaultLanguage
}

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

func WithContext(ctx context.Context, lang Language) context.Context {
	return context.WithValue(ctx, langContextKey, lang)
}

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

func T(lang Language, key string, args ...any) string {
	langDict, ok := messages[lang]
	if !ok {
		langDict = messages[DefaultLanguage]
	}

	msg, exists := langDict[key]
	if !exists {
		if lang != DefaultLanguage {
			if defaultDict, ok := messages[DefaultLanguage]; ok {
				msg = defaultDict[key]
			}
		}
	}

	if msg == "" {
		msg = key
	}

	if len(args) > 0 {
		return fmt.Sprintf(msg, args...)
	}
	return msg
}

func TCtx(ctx context.Context, key string, args ...any) string {
	return T(FromContext(ctx), key, args...)
}

func RegisterMessages(lang Language, newMessages map[string]string) {
	if _, ok := messages[lang]; !ok {
		messages[lang] = make(map[string]string)
	}
	for k, v := range newMessages {
		messages[lang][k] = v
	}
}
