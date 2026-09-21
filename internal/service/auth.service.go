package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"chatapp/internal/config"
	"chatapp/internal/dto"
	"chatapp/internal/model"
	"chatapp/internal/repository"
	"chatapp/pkg/jwt"
	"chatapp/pkg/mailer"
)

type AuthService struct {
	authRepo   repository.AuthRepository
	deviceRepo repository.DeviceRepository
	mailer     mailer.Mailer
	cfg        *config.Config
}

func NewAuthService(
	authRepo repository.AuthRepository,
	deviceRepo repository.DeviceRepository,
	mailer mailer.Mailer,
	cfg *config.Config,
) *AuthService {
	return &AuthService{
		authRepo:   authRepo,
		deviceRepo: deviceRepo,
		mailer:     mailer,
		cfg:        cfg,
	}
}

// Generate secure 6-digit OTP code
func generateOTP() (string, error) {
	var table = [...]byte{'1', '2', '3', '4', '5', '6', '7', '8', '9', '0'}
	b := make([]byte, 6)
	n, err := io.ReadAtLeast(rand.Reader, b, 6)
	if n != 6 || err != nil {
		return "", err
	}
	for i := 0; i < len(b); i++ {
		b[i] = table[int(b[i])%len(table)]
	}
	return string(b), nil
}

// Register creates a new account, user, and registers the initial primary device
func (s *AuthService) Register(ctx context.Context, req dto.RegisterRequest, ipAddress, userAgent string) (*dto.AuthResponse, error) {
	req.Phone = strings.TrimSpace(req.Phone)
	req.Mail = strings.TrimSpace(strings.ToLower(req.Mail))
	req.FullName = strings.TrimSpace(req.FullName)
	req.DisplayName = strings.TrimSpace(req.DisplayName)

	if !phonePattern.MatchString(req.Phone) {
		return nil, fmt.Errorf("số điện thoại không hợp lệ")
	}
	if req.Mail != "" {
		if _, err := mail.ParseAddress(req.Mail); err != nil {
			return nil, fmt.Errorf("email không hợp lệ")
		}
	}
	if len(req.Password) < 8 {
		return nil, fmt.Errorf("mật khẩu phải có ít nhất 8 ký tự")
	}

	// Kiểm tra trùng phone
	existingPhone, err := s.authRepo.FindAccountByPhone(ctx, req.Phone)
	if err == nil && existingPhone != nil {
		return nil, fmt.Errorf("số điện thoại đã được sử dụng")
	}
	if req.Mail != "" {
		existingMail, err := s.authRepo.FindAccountByMail(ctx, req.Mail)
		if err == nil && existingMail != nil {
			return nil, fmt.Errorf("email đã được sử dụng")
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("không thể mã hóa mật khẩu: %w", err)
	}

	account := &model.Account{
		Phone:        req.Phone,
		PasswordHash: string(hash),
		HashAlgo:     "bcrypt",
		IsVerified:   true,
	}
	if req.Mail != "" {
		account.Mail = &req.Mail
	}

	user := &model.User{
		FullName:    req.FullName,
		DisplayName: req.DisplayName,
		Status:      "offline",
	}

	deviceID := strings.TrimSpace(req.DeviceID)
	if deviceID == "" {
		deviceID = uuid.New().String()
	}
	deviceName := strings.TrimSpace(req.DeviceName)
	if deviceName == "" {
		deviceName = "Thiết bị chính"
	}
	deviceType := strings.TrimSpace(req.DeviceType)
	if deviceType == "" {
		deviceType = "web"
	}

	// Thiết bị đầu tiên đăng ký luôn là thiết bị chính (Primary)
	session := &model.DeviceSession{
		DeviceID:   deviceID,
		DeviceName: deviceName,
		DeviceType: deviceType,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		IsPrimary:  true,
		IsActive:   true,
	}

	if err := s.authRepo.CreateAccountAndUser(ctx, account, user, session); err != nil {
		return nil, fmt.Errorf("lỗi tạo tài khoản: %w", err)
	}

	// Tạo tokens
	accessToken, err := jwt.GenerateSessionToken(
		user,
		account.ID.String(),
		session.ID.String(),
		session.DeviceID,
		session.IsPrimary,
		s.cfg.JWTSecret,
		24*time.Hour,
	)
	if err != nil {
		return nil, err
	}

	refreshToken, err := jwt.GenerateSessionRefreshToken(
		user,
		account.ID.String(),
		session.ID.String(),
		session.DeviceID,
		session.IsPrimary,
		s.cfg.JWTSecret,
		7*24*time.Hour,
	)
	if err != nil {
		return nil, err
	}

	// Lưu refresh token vào session
	session.RefreshToken = refreshToken
	_ = s.deviceRepo.UpdateSession(ctx, session)

	return &dto.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         toUserResponseDTO(user),
		Session:      toDeviceSessionResponseDTO(session, true),
	}, nil
}

// Login handles multi-device login, checks 2FA status, and tracks primary device
func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest, ipAddress, userAgent string) (*dto.AuthResponse, error) {
	identifier := strings.TrimSpace(req.Identifier)
	if identifier == "" {
		return nil, fmt.Errorf("vui lòng nhập email hoặc số điện thoại")
	}

	account, err := s.authRepo.FindAccountByIdentifier(ctx, identifier)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("tài khoản hoặc mật khẩu không chính xác")
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errors.New("tài khoản hoặc mật khẩu không chính xác")
	}

	// Kiểm tra nếu tài khoản BẬT XÁC THỰC 2 BƯỚC (2FA qua email)
	if account.TwoFactorEnabled {
		if account.Mail == nil || *account.Mail == "" {
			return nil, errors.New("tài khoản chưa cấu hình email để nhận mã xác thực 2 bước")
		}

		otpCode, err := generateOTP()
		if err != nil {
			return nil, fmt.Errorf("không thể tạo mã xác thực: %w", err)
		}

		vCode := &model.VerificationCode{
			AccountId: &account.ID,
			Target:    *account.Mail,
			Code:      otpCode,
			Type:      "2fa_login",
			ExpiresAt: time.Now().Add(5 * time.Minute),
			IsUsed:    false,
		}
		if err := s.authRepo.CreateVerificationCode(ctx, vCode); err != nil {
			return nil, err
		}

		// Gửi email OTP
		_ = s.mailer.SendOTP(*account.Mail, otpCode, "2fa_login")

		// Sinh Temp Token dùng để gọi verify 2FA
		tempToken, err := jwt.GenerateTempTwoFactorToken(account.ID.String(), *account.Mail, s.cfg.JWTSecret, 10*time.Minute)
		if err != nil {
			return nil, err
		}

		return &dto.AuthResponse{
			TwoFactorRequired: true,
			TempToken:         tempToken,
		}, nil
	}

	// Nếu không có 2FA, hoàn tất đăng nhập thiết bị
	return s.completeDeviceLogin(ctx, account, req.DeviceID, req.DeviceName, req.DeviceType, ipAddress, userAgent)
}

// VerifyLogin2FA verifies the 2FA OTP code and completes login
func (s *AuthService) VerifyLogin2FA(ctx context.Context, req dto.Verify2FARequest, ipAddress, userAgent string) (*dto.AuthResponse, error) {
	claims, err := jwt.ValidateTempTwoFactorToken(req.TempToken, s.cfg.JWTSecret)
	if err != nil {
		return nil, errors.New("phiên xác thực 2 bước không hợp lệ hoặc đã hết hạn")
	}

	accountIDStr := claims.AccountID
	if accountIDStr == "" {
		accountIDStr = claims.UserID
	}
	accountUUID, err := uuid.Parse(accountIDStr)
	if err != nil {
		return nil, errors.New("account ID không hợp lệ")
	}

	account, err := s.authRepo.FindAccountByID(ctx, accountUUID)
	if err != nil {
		return nil, errors.New("tài khoản không tồn tại")
	}

	if account.Mail == nil {
		return nil, errors.New("email tài khoản không tồn tại")
	}

	// Kiểm tra mã OTP
	vc, err := s.authRepo.FindValidVerificationCode(ctx, *account.Mail, req.OTPCode, "2fa_login")
	if err != nil {
		return nil, errors.New("mã xác thực không hợp lệ hoặc đã hết hạn")
	}

	_ = s.authRepo.MarkVerificationCodeUsed(ctx, vc.ID)

	return s.completeDeviceLogin(ctx, account, req.DeviceID, req.DeviceName, req.DeviceType, ipAddress, userAgent)
}

// Helper: hoàn tất đăng nhập cho thiết bị (tạo mới hoặc cập nhật session, cấp token)
func (s *AuthService) completeDeviceLogin(
	ctx context.Context,
	account *model.Account,
	deviceID, deviceName, deviceType, ipAddress, userAgent string,
) (*dto.AuthResponse, error) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		deviceID = uuid.New().String()
	}
	if deviceName == "" {
		deviceName = "Thiết bị chưa đặt tên"
	}
	if deviceType == "" {
		deviceType = "web"
	}

	// Kiểm tra xem thiết bị này đã từng đăng nhập chưa
	session, err := s.deviceRepo.FindSessionByDeviceID(ctx, account.ID, deviceID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// Kiểm tra xem tài khoản hiện tại đã có thiết bị chính (active) nào chưa
	primaryCount, err := s.deviceRepo.CountActivePrimarySessions(ctx, account.ID)
	if err != nil {
		return nil, err
	}

	if session == nil {
		// Chưa có session -> tạo mới
		// Nếu chưa có thiết bị chính nào -> gán thiết bị này làm primary
		isPrimary := (primaryCount == 0)
		session = &model.DeviceSession{
			AccountId:  account.ID,
			DeviceID:   deviceID,
			DeviceName: deviceName,
			DeviceType: deviceType,
			IPAddress:  ipAddress,
			UserAgent:  userAgent,
			IsPrimary:  isPrimary,
			IsActive:   true,
		}
		if err := s.deviceRepo.CreateSession(ctx, session); err != nil {
			return nil, err
		}
	} else {
		// Đã có session -> kích hoạt lại và update thông tin mới nhất
		session.IsActive = true
		session.DeviceName = deviceName
		session.DeviceType = deviceType
		session.IPAddress = ipAddress
		session.UserAgent = userAgent
		session.LastActiveAt = time.Now()

		// Nếu tài khoản không còn thiết bị chính active nào khác -> khôi phục primary cho thiết bị này
		if primaryCount == 0 {
			session.IsPrimary = true
		}

		if err := s.deviceRepo.UpdateSession(ctx, session); err != nil {
			return nil, err
		}
	}

	// Cập nhật last_login_at cho user
	user := &account.User
	now := time.Now()
	user.LastLoginAt = &now
	_ = s.authRepo.CreateAccountAndUser(ctx, account, user, nil) // update via gorm

	// Tạo Access Token & Refresh Token
	accessToken, err := jwt.GenerateSessionToken(
		user,
		account.ID.String(),
		session.ID.String(),
		session.DeviceID,
		session.IsPrimary,
		s.cfg.JWTSecret,
		24*time.Hour,
	)
	if err != nil {
		return nil, err
	}

	refreshToken, err := jwt.GenerateSessionRefreshToken(
		user,
		account.ID.String(),
		session.ID.String(),
		session.DeviceID,
		session.IsPrimary,
		s.cfg.JWTSecret,
		7*24*time.Hour,
	)
	if err != nil {
		return nil, err
	}

	session.RefreshToken = refreshToken
	_ = s.deviceRepo.UpdateSession(ctx, session)

	return &dto.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         toUserResponseDTO(user),
		Session:      toDeviceSessionResponseDTO(session, true),
	}, nil
}

// RefreshToken issues a new access token if session is still active
func (s *AuthService) RefreshToken(ctx context.Context, refreshTokenStr string) (*dto.AuthResponse, error) {
	claims, err := jwt.ValidateToken(refreshTokenStr, s.cfg.JWTSecret)
	if err != nil {
		return nil, errors.New("refresh token không hợp lệ hoặc đã hết hạn")
	}

	if claims.TokenType != jwt.RefreshTokenType {
		return nil, errors.New("loại token không hợp lệ")
	}

	sessionUUID, err := uuid.Parse(claims.SessionID)
	if err != nil {
		return nil, errors.New("phiên không hợp lệ")
	}

	session, err := s.deviceRepo.FindSessionByID(ctx, sessionUUID)
	if err != nil || !session.IsActive {
		return nil, errors.New("phiên thiết bị đã bị đăng xuất hoặc thu hồi")
	}

	accountUUID, err := uuid.Parse(claims.AccountID)
	if err != nil {
		return nil, errors.New("account id không hợp lệ")
	}

	account, err := s.authRepo.FindAccountByID(ctx, accountUUID)
	if err != nil {
		return nil, errors.New("tài khoản không tồn tại")
	}

	user := &account.User

	newAccessToken, err := jwt.GenerateSessionToken(
		user,
		account.ID.String(),
		session.ID.String(),
		session.DeviceID,
		session.IsPrimary,
		s.cfg.JWTSecret,
		24*time.Hour,
	)
	if err != nil {
		return nil, err
	}

	return &dto.AuthResponse{
		AccessToken:  newAccessToken,
		RefreshToken: refreshTokenStr,
		User:         toUserResponseDTO(user),
		Session:      toDeviceSessionResponseDTO(session, true),
	}, nil
}

// Logout deactivates current session
func (s *AuthService) Logout(ctx context.Context, sessionIDStr string) error {
	sessionUUID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		return errors.New("session id không hợp lệ")
	}
	return s.deviceRepo.RevokeSession(ctx, sessionUUID)
}

// ForgotPassword generates OTP for password reset
func (s *AuthService) ForgotPassword(ctx context.Context, email string) error {
	email = strings.TrimSpace(strings.ToLower(email))
	account, err := s.authRepo.FindAccountByMail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Để tránh user enumeration, vẫn trả về thành công hoặc báo không tìm thấy
			return errors.New("không tìm thấy tài khoản với email này")
		}
		return err
	}

	otpCode, err := generateOTP()
	if err != nil {
		return err
	}

	vc := &model.VerificationCode{
		AccountId: &account.ID,
		Target:    email,
		Code:      otpCode,
		Type:      "forgot_password",
		ExpiresAt: time.Now().Add(10 * time.Minute),
		IsUsed:    false,
	}
	if err := s.authRepo.CreateVerificationCode(ctx, vc); err != nil {
		return err
	}

	return s.mailer.SendOTP(email, otpCode, "forgot_password")
}

// ResetPassword resets account password using verified OTP
func (s *AuthService) ResetPassword(ctx context.Context, req dto.ResetPasswordRequest) error {
	email := strings.TrimSpace(strings.ToLower(req.Email))
	account, err := s.authRepo.FindAccountByMail(ctx, email)
	if err != nil {
		return errors.New("tài khoản không tồn tại")
	}

	vc, err := s.authRepo.FindValidVerificationCode(ctx, email, req.OTPCode, "forgot_password")
	if err != nil {
		return errors.New("mã OTP không chính xác hoặc đã hết hạn")
	}

	if len(req.NewPassword) < 8 {
		return errors.New("mật khẩu mới phải có ít nhất 8 ký tự")
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	account.PasswordHash = string(newHash)
	if err := s.authRepo.UpdateAccount(ctx, account); err != nil {
		return err
	}

	_ = s.authRepo.MarkVerificationCodeUsed(ctx, vc.ID)
	return nil
}

// ======================== QUẢN LÝ THIẾT BỊ ========================

// GetDevices returns all active device sessions for the account
func (s *AuthService) GetDevices(ctx context.Context, accountIDStr, currentSessionIDStr string) ([]dto.DeviceSessionResponse, error) {
	accountUUID, err := uuid.Parse(accountIDStr)
	if err != nil {
		return nil, errors.New("account ID không hợp lệ")
	}

	sessions, err := s.deviceRepo.ListSessionsByAccountID(ctx, accountUUID)
	if err != nil {
		return nil, err
	}

	res := make([]dto.DeviceSessionResponse, len(sessions))
	for i, sess := range sessions {
		isCurrent := (sess.ID.String() == currentSessionIDStr)
		res[i] = *toDeviceSessionResponseDTO(&sess, isCurrent)
	}
	return res, nil
}

// LogoutDevice: CHỈ THIẾT BỊ CHÍNH (Primary Device) mới có quyền đăng xuất thiết bị khác
func (s *AuthService) LogoutDevice(ctx context.Context, accountIDStr, currentSessionIDStr, targetSessionIDStr string, isPrimary bool) error {
	if !isPrimary {
		return errors.New("chỉ thiết bị chính mới có quyền đăng xuất tài khoản từ thiết bị khác")
	}

	if currentSessionIDStr == targetSessionIDStr {
		return errors.New("không thể thu hồi thiết bị hiện tại bằng chức năng này, vui lòng sử dụng đăng xuất thông thường")
	}

	targetSessionUUID, err := uuid.Parse(targetSessionIDStr)
	if err != nil {
		return errors.New("target session ID không hợp lệ")
	}

	session, err := s.deviceRepo.FindSessionByID(ctx, targetSessionUUID)
	if err != nil || session.AccountId.String() != accountIDStr {
		return errors.New("thiết bị cần đăng xuất không tồn tại")
	}

	return s.deviceRepo.RevokeSession(ctx, targetSessionUUID)
}

// LogoutAllOtherDevices: CHỈ THIẾT BỊ CHÍNH mới có quyền đăng xuất TẤT CẢ các thiết bị khác
func (s *AuthService) LogoutAllOtherDevices(ctx context.Context, accountIDStr, currentSessionIDStr string, isPrimary bool) error {
	if !isPrimary {
		return errors.New("chỉ thiết bị chính mới có quyền đăng xuất tất cả các thiết bị khác")
	}

	accountUUID, err := uuid.Parse(accountIDStr)
	if err != nil {
		return errors.New("account ID không hợp lệ")
	}

	currentSessionUUID, err := uuid.Parse(currentSessionIDStr)
	if err != nil {
		return errors.New("current session ID không hợp lệ")
	}

	return s.deviceRepo.RevokeOtherSessions(ctx, accountUUID, currentSessionUUID)
}

// TransferPrimary: Chuyển quyền thiết bị chính cho một thiết bị khác (yêu cầu mật khẩu)
func (s *AuthService) TransferPrimary(ctx context.Context, accountIDStr, targetSessionIDStr, password string, isPrimary bool) error {
	if !isPrimary {
		return errors.New("chỉ thiết bị chính hiện tại mới có quyền chuyển giao quyền thiết bị chính")
	}

	accountUUID, err := uuid.Parse(accountIDStr)
	if err != nil {
		return errors.New("account ID không hợp lệ")
	}

	account, err := s.authRepo.FindAccountByID(ctx, accountUUID)
	if err != nil {
		return errors.New("tài khoản không tồn tại")
	}

	// Xác thực mật khẩu
	if err := bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(password)); err != nil {
		return errors.New("mật khẩu xác nhận không chính xác")
	}

	targetSessionUUID, err := uuid.Parse(targetSessionIDStr)
	if err != nil {
		return errors.New("target session ID không hợp lệ")
	}

	targetSession, err := s.deviceRepo.FindSessionByID(ctx, targetSessionUUID)
	if err != nil || targetSession.AccountId != accountUUID {
		return errors.New("thiết bị mục tiêu không tồn tại")
	}

	return s.deviceRepo.SetPrimarySession(ctx, accountUUID, targetSessionUUID)
}

// ======================== QUẢN LÝ 2FA (EMAIL) ========================

// RequestEnable2FA: CHỈ THIẾT BỊ CHÍNH mới có quyền gửi yêu cầu bật 2FA
func (s *AuthService) RequestEnable2FA(ctx context.Context, accountIDStr string, isPrimary bool) error {
	if !isPrimary {
		return errors.New("chỉ thiết bị chính mới có quyền kích hoạt tính năng xác thực 2 bước")
	}

	accountUUID, err := uuid.Parse(accountIDStr)
	if err != nil {
		return errors.New("account ID không hợp lệ")
	}

	account, err := s.authRepo.FindAccountByID(ctx, accountUUID)
	if err != nil {
		return errors.New("tài khoản không tồn tại")
	}

	if account.Mail == nil || *account.Mail == "" {
		return errors.New("tài khoản cần cập nhật email hợp lệ trước khi bật xác thực 2 bước")
	}

	if account.TwoFactorEnabled {
		return errors.New("tính năng xác thực 2 bước đã được kích hoạt trước đó")
	}

	otpCode, err := generateOTP()
	if err != nil {
		return err
	}

	vc := &model.VerificationCode{
		AccountId: &account.ID,
		Target:    *account.Mail,
		Code:      otpCode,
		Type:      "2fa_enable",
		ExpiresAt: time.Now().Add(5 * time.Minute),
		IsUsed:    false,
	}
	if err := s.authRepo.CreateVerificationCode(ctx, vc); err != nil {
		return err
	}

	return s.mailer.SendOTP(*account.Mail, otpCode, "2fa_enable")
}

// ConfirmEnable2FA: CHỈ THIẾT BỊ CHÍNH xác nhận mã OTP để bật 2FA
func (s *AuthService) ConfirmEnable2FA(ctx context.Context, accountIDStr, otpCode string, isPrimary bool) error {
	if !isPrimary {
		return errors.New("chỉ thiết bị chính mới có quyền kích hoạt tính năng xác thực 2 bước")
	}

	accountUUID, err := uuid.Parse(accountIDStr)
	if err != nil {
		return errors.New("account ID không hợp lệ")
	}

	account, err := s.authRepo.FindAccountByID(ctx, accountUUID)
	if err != nil {
		return errors.New("tài khoản không tồn tại")
	}

	if account.Mail == nil {
		return errors.New("email tài khoản không tồn tại")
	}

	vc, err := s.authRepo.FindValidVerificationCode(ctx, *account.Mail, otpCode, "2fa_enable")
	if err != nil {
		return errors.New("mã xác thực không hợp lệ hoặc đã hết hạn")
	}

	account.TwoFactorEnabled = true
	if err := s.authRepo.UpdateAccount(ctx, account); err != nil {
		return err
	}

	_ = s.authRepo.MarkVerificationCodeUsed(ctx, vc.ID)
	return nil
}

// Disable2FA: CHỈ THIẾT BỊ CHÍNH mới có quyền tắt 2FA (cần mật khẩu)
func (s *AuthService) Disable2FA(ctx context.Context, accountIDStr, password string, isPrimary bool) error {
	if !isPrimary {
		return errors.New("chỉ thiết bị chính mới có quyền tắt tính năng xác thực 2 bước")
	}

	accountUUID, err := uuid.Parse(accountIDStr)
	if err != nil {
		return errors.New("account ID không hợp lệ")
	}

	account, err := s.authRepo.FindAccountByID(ctx, accountUUID)
	if err != nil {
		return errors.New("tài khoản không tồn tại")
	}

	if !account.TwoFactorEnabled {
		return errors.New("tính năng xác thực 2 bước hiện chưa được bật")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(password)); err != nil {
		return errors.New("mật khẩu xác nhận không chính xác")
	}

	account.TwoFactorEnabled = false
	return s.authRepo.UpdateAccount(ctx, account)
}

// Helpers
func toUserResponseDTO(u *model.User) *dto.UserResponse {
	if u == nil {
		return nil
	}
	res := &dto.UserResponse{
		ID:          u.ID.String(),
		FullName:    u.FullName,
		DisplayName: u.DisplayName,
		AvatarUrl:   u.AvatarUrl,
		CoverUrl:    u.CoverUrl,
		Bio:         u.Bio,
		Status:      u.Status,
		CreatedAt:   u.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   u.UpdatedAt.Format(time.RFC3339),
	}
	if u.LastLoginAt != nil {
		t := u.LastLoginAt.Format(time.RFC3339)
		res.LastLoginAt = &t
	}
	return res
}

func toDeviceSessionResponseDTO(s *model.DeviceSession, isCurrent bool) *dto.DeviceSessionResponse {
	if s == nil {
		return nil
	}
	return &dto.DeviceSessionResponse{
		ID:           s.ID.String(),
		DeviceID:     s.DeviceID,
		DeviceName:   s.DeviceName,
		DeviceType:   s.DeviceType,
		IPAddress:    s.IPAddress,
		IsPrimary:    s.IsPrimary,
		IsActive:     s.IsActive,
		IsCurrent:    isCurrent,
		LastActiveAt: s.LastActiveAt.Format(time.RFC3339),
		CreatedAt:    s.CreatedAt.Format(time.RFC3339),
	}
}
