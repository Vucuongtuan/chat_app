package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"chatapp/internal/config"
	"chatapp/internal/dto"
	"chatapp/internal/model"
	"chatapp/internal/repository"
	"chatapp/pkg/eventbus"
	"chatapp/pkg/jwt"
	"chatapp/pkg/mailer"
)

var authPhonePattern = regexp.MustCompile(`^\+?[0-9]{7,15}$`)

type AuthService struct {
	authRepo   repository.AuthRepository
	deviceRepo repository.DeviceRepository
	mailer     mailer.Mailer
	cfg        *config.Config
	bus        *eventbus.Bus
}

func NewAuthService(
	authRepo repository.AuthRepository,
	deviceRepo repository.DeviceRepository,
	mailer mailer.Mailer,
	cfg *config.Config,
	buses ...*eventbus.Bus,
) *AuthService {
	var bus *eventbus.Bus
	if len(buses) > 0 {
		bus = buses[0]
	}
	return &AuthService{
		authRepo:   authRepo,
		deviceRepo: deviceRepo,
		mailer:     mailer,
		cfg:        cfg,
		bus:        bus,
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

// normalizePlatform chuẩn hoá và validate platform string
func normalizePlatform(p string) string {
	p = strings.ToLower(strings.TrimSpace(p))
	switch p {
	case model.PlatformPhone, model.PlatformTablet, model.PlatformPC, model.PlatformLaptop, model.PlatformWeb:
		return p
	default:
		return model.PlatformWeb
	}
}

func requestPlatform(platform, legacyDeviceType string) string {
	if strings.TrimSpace(platform) == "" {
		// The old device_type field did not participate in device-role rules.
		// Keep it as a secondary/web device; only the explicit platform=phone
		// enables the single-primary-phone behavior.
		return model.PlatformWeb
	}
	return normalizePlatform(platform)
}

// Register creates a new account, user, and registers the initial primary device
func (s *AuthService) Register(ctx context.Context, req dto.RegisterRequest, ipAddress, userAgent string) (*dto.AuthResponse, error) {
	req.Phone = strings.TrimSpace(req.Phone)
	req.Mail = strings.TrimSpace(strings.ToLower(req.Mail))
	req.FullName = strings.TrimSpace(req.FullName)
	req.DisplayName = strings.TrimSpace(req.DisplayName)

	if !authPhonePattern.MatchString(req.Phone) {
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
	platform := requestPlatform(req.Platform, req.DeviceType)

	// Thiết bị đầu tiên đăng ký luôn là thiết bị chính (Primary)
	// Nếu là phone → primary, nếu không phải phone → vẫn primary vì là thiết bị đầu tiên
	session := &model.DeviceSession{
		DeviceID:   deviceID,
		DeviceName: deviceName,
		Platform:   platform,
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
	return s.completeDeviceLogin(ctx, account, req.DeviceID, req.DeviceName, requestPlatform(req.Platform, req.DeviceType), ipAddress, userAgent)
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

	return s.completeDeviceLogin(ctx, account, req.DeviceID, req.DeviceName, req.Platform, ipAddress, userAgent)
}

// completeDeviceLogin hoàn tất đăng nhập cho thiết bị với các rule:
//   - Phone: chỉ 1 session active duy nhất → revoke phone cũ trước khi tạo mới
//   - Secondary: tổng active ≤ MaxSecondaryDevices
func (s *AuthService) completeDeviceLogin(
	ctx context.Context,
	account *model.Account,
	deviceID, deviceName, platform, ipAddress, userAgent string,
) (*dto.AuthResponse, error) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		deviceID = uuid.New().String()
	}
	if deviceName == "" {
		deviceName = "Thiết bị chưa đặt tên"
	}
	platform = normalizePlatform(platform)

	// ── Enforce giới hạn thiết bị ──────────────────────────────────────────
	if model.IsPhonePlatform(platform) {
		// Phone: chỉ 1 session active → revoke tất cả phone cũ trước
		phoneCount, err := s.deviceRepo.CountActivePhoneSessions(ctx, account.ID)
		if err != nil {
			return nil, err
		}
		if phoneCount > 0 {
			// Kiểm tra xem deviceID này có phải phone đang active không
			existingSession, err := s.deviceRepo.FindSessionByDeviceID(ctx, account.ID, deviceID)
			if err != nil || existingSession.Platform != model.PlatformPhone {
				// Phone khác đang login → revoke hết
				if err := s.deviceRepo.RevokePhoneSessions(ctx, account.ID); err != nil {
					return nil, err
				}
				// Phát event để notify phone cũ bị đăng xuất
				if s.bus != nil {
					s.bus.Publish(eventbus.Event{
						Type:    eventbus.EventDevicePhoneReplaced,
						Payload: map[string]interface{}{"account_id": account.ID.String()},
					})
				}
			}
		}
	} else {
		// Secondary: kiểm tra tổng giới hạn MaxSecondaryDevices
		// Bỏ qua nếu device đã tồn tại (re-login)
		existingSession, existErr := s.deviceRepo.FindSessionByDeviceID(ctx, account.ID, deviceID)
		isNewDevice := existErr != nil || existingSession == nil

		if isNewDevice {
			secCount, err := s.deviceRepo.CountActiveSecondarySessions(ctx, account.ID)
			if err != nil {
				return nil, err
			}
			if secCount >= model.MaxSecondaryDevices {
				return nil, fmt.Errorf("đã đạt giới hạn tối đa %d thiết bị đang đăng nhập, vui lòng đăng xuất thiết bị khác trước", model.MaxSecondaryDevices)
			}
		}
	}

	// ── Tạo hoặc cập nhật session ───────────────────────────────────────────
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
		// Chưa có session → tạo mới
		// Phone là primary; các thiết bị khác chỉ primary nếu chưa có ai primary
		isPrimary := model.IsPhonePlatform(platform) || primaryCount == 0
		session = &model.DeviceSession{
			AccountId:  account.ID,
			DeviceID:   deviceID,
			DeviceName: deviceName,
			Platform:   platform,
			IPAddress:  ipAddress,
			UserAgent:  userAgent,
			IsPrimary:  isPrimary,
			IsActive:   true,
		}
		if err := s.deviceRepo.CreateSession(ctx, session); err != nil {
			return nil, err
		}
	} else {
		// Đã có session → kích hoạt lại và update thông tin mới nhất
		session.IsActive = true
		session.DeviceName = deviceName
		session.Platform = platform
		session.IPAddress = ipAddress
		session.UserAgent = userAgent
		session.LastActiveAt = time.Now()

		// Khôi phục primary nếu không còn ai là primary
		if primaryCount == 0 || model.IsPhonePlatform(platform) {
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

// ======================== QR LOGIN ========================

// QRChallenge tạo QR token cho secondary device
func (s *AuthService) QRChallenge(ctx context.Context, req dto.QRChallengeRequest, ipAddress, userAgent string) (*dto.QRChallengeResponse, error) {
	platform := normalizePlatform(req.Platform)
	if model.IsPhonePlatform(platform) {
		return nil, errors.New("không thể dùng QR login cho thiết bị phone")
	}

	deviceID := strings.TrimSpace(req.DeviceID)
	if deviceID == "" {
		deviceID = uuid.New().String()
	}

	// Tạo 32-byte random hex token (64 chars)
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("không thể tạo QR token: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)

	expiresAt := time.Now().Add(5 * time.Minute)
	q := &model.QRLoginSession{
		Token:      token,
		DeviceID:   deviceID,
		DeviceName: strings.TrimSpace(req.DeviceName),
		Platform:   platform,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		Status:     "pending",
		ExpiresAt:  expiresAt,
	}

	if err := s.deviceRepo.CreateQRSession(ctx, q); err != nil {
		return nil, err
	}

	return &dto.QRChallengeResponse{
		Token:     token,
		ExpiresAt: expiresAt.Format(time.RFC3339),
	}, nil
}

// QRStatus được secondary device poll để biết kết quả QR login
func (s *AuthService) QRStatus(ctx context.Context, token string) (*dto.QRStatusResponse, error) {
	q, err := s.deviceRepo.FindQRSession(ctx, token)
	if err != nil {
		return nil, errors.New("QR token không tồn tại")
	}

	// Kiểm tra expired
	if time.Now().After(q.ExpiresAt) && q.Status == "pending" {
		return &dto.QRStatusResponse{Status: "expired"}, nil
	}

	if q.Status != "approved" {
		return &dto.QRStatusResponse{Status: q.Status}, nil
	}

	// Đã approved → lấy session và tạo tokens
	if q.SessionID == nil || q.AccountId == nil {
		return nil, errors.New("phiên QR không hợp lệ")
	}

	session, err := s.deviceRepo.FindSessionByID(ctx, *q.SessionID)
	if err != nil {
		return nil, errors.New("không tìm thấy session")
	}

	account, err := s.authRepo.FindAccountByID(ctx, *q.AccountId)
	if err != nil {
		return nil, errors.New("tài khoản không tồn tại")
	}

	user := &account.User
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

	return &dto.QRStatusResponse{
		Status:       "approved",
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// QRApprove được primary phone gọi sau khi scan QR để approve đăng nhập
func (s *AuthService) QRApprove(ctx context.Context, accountIDStr, token string) error {
	q, err := s.deviceRepo.FindQRSession(ctx, token)
	if err != nil {
		return errors.New("QR token không tồn tại")
	}

	if q.Status != "pending" {
		return fmt.Errorf("QR token đã ở trạng thái '%s'", q.Status)
	}

	if time.Now().After(q.ExpiresAt) {
		return errors.New("QR token đã hết hạn")
	}

	accountUUID, err := uuid.Parse(accountIDStr)
	if err != nil {
		return errors.New("account ID không hợp lệ")
	}

	account, err := s.authRepo.FindAccountByID(ctx, accountUUID)
	if err != nil {
		return errors.New("tài khoản không tồn tại")
	}

	// Kiểm tra giới hạn secondary
	secCount, err := s.deviceRepo.CountActiveSecondarySessions(ctx, accountUUID)
	if err != nil {
		return err
	}
	if secCount >= model.MaxSecondaryDevices {
		return fmt.Errorf("đã đạt giới hạn %d thiết bị, vui lòng đăng xuất thiết bị khác", model.MaxSecondaryDevices)
	}

	// Tạo session cho secondary device
	primaryCount, _ := s.deviceRepo.CountActivePrimarySessions(ctx, accountUUID)
	session := &model.DeviceSession{
		AccountId:  accountUUID,
		DeviceID:   q.DeviceID,
		DeviceName: q.DeviceName,
		Platform:   q.Platform,
		IPAddress:  q.IPAddress,
		UserAgent:  q.UserAgent,
		IsPrimary:  primaryCount == 0,
		IsActive:   true,
	}

	// Nếu device đã có session → cập nhật thay vì tạo mới
	existing, err := s.deviceRepo.FindSessionByDeviceID(ctx, accountUUID, q.DeviceID)
	if err == nil && existing != nil {
		existing.IsActive = true
		existing.DeviceName = q.DeviceName
		existing.Platform = q.Platform
		existing.IPAddress = q.IPAddress
		existing.UserAgent = q.UserAgent
		if err := s.deviceRepo.UpdateSession(ctx, existing); err != nil {
			return err
		}
		session = existing
	} else {
		if err := s.deviceRepo.CreateSession(ctx, session); err != nil {
			return err
		}
	}

	_ = account // used for context

	return s.deviceRepo.ApproveQRSession(ctx, token, accountUUID, session.ID)
}

// QRReject được primary phone gọi để từ chối QR login
func (s *AuthService) QRReject(ctx context.Context, token string) error {
	q, err := s.deviceRepo.FindQRSession(ctx, token)
	if err != nil {
		return errors.New("QR token không tồn tại")
	}
	if q.Status != "pending" {
		return fmt.Errorf("QR token đã ở trạng thái '%s'", q.Status)
	}
	return s.deviceRepo.RejectQRSession(ctx, token)
}

// ======================== SECONDARY OTP LOGIN ========================

// SecondaryRequestOTP: secondary device xin OTP, server push OTP đến primary phone
func (s *AuthService) SecondaryRequestOTP(ctx context.Context, req dto.SecondaryOTPRequest, ipAddress, userAgent string) (*dto.SecondaryOTPResponse, error) {
	platform := normalizePlatform(req.Platform)
	if model.IsPhonePlatform(platform) {
		return nil, errors.New("không thể dùng OTP login cho thiết bị phone")
	}

	phone := strings.TrimSpace(req.Phone)
	account, err := s.authRepo.FindAccountByPhone(ctx, phone)
	if err != nil {
		return nil, errors.New("không tìm thấy tài khoản với số điện thoại này")
	}

	// Kiểm tra giới hạn secondary trước
	secCount, err := s.deviceRepo.CountActiveSecondarySessions(ctx, account.ID)
	if err != nil {
		return nil, err
	}
	if secCount >= model.MaxSecondaryDevices {
		return nil, fmt.Errorf("đã đạt giới hạn %d thiết bị, vui lòng đăng xuất thiết bị khác", model.MaxSecondaryDevices)
	}

	otpCode, err := generateOTP()
	if err != nil {
		return nil, fmt.Errorf("không thể tạo OTP: %w", err)
	}

	deviceID := strings.TrimSpace(req.DeviceID)
	if deviceID == "" {
		deviceID = uuid.New().String()
	}

	expiresAt := time.Now().Add(3 * time.Minute)
	otpSession := &model.SecondaryOTPSession{
		AccountId:  account.ID,
		OTPCode:    otpCode,
		DeviceID:   deviceID,
		DeviceName: strings.TrimSpace(req.DeviceName),
		Platform:   platform,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		IsUsed:     false,
		ExpiresAt:  expiresAt,
	}

	if err := s.deviceRepo.CreateSecondaryOTPSession(ctx, otpSession); err != nil {
		return nil, err
	}

	// Push OTP đến primary phone qua eventbus (WebSocket handler lắng nghe và forward)
	if s.bus != nil {
		s.bus.Publish(eventbus.Event{
			Type: eventbus.EventDeviceSecondaryOTPRequested,
			Payload: map[string]interface{}{
				"account_id":     account.ID.String(),
				"otp_code":       otpCode,
				"device_name":    req.DeviceName,
				"platform":       platform,
				"otp_session_id": otpSession.ID.String(),
				"expires_at":     expiresAt.Format(time.RFC3339),
			},
		})
	}

	return &dto.SecondaryOTPResponse{
		OTPSessionID: otpSession.ID.String(),
		ExpiresAt:    expiresAt.Format(time.RFC3339),
	}, nil
}

// SecondaryVerifyOTP: secondary device nhập OTP để hoàn tất đăng nhập
func (s *AuthService) SecondaryVerifyOTP(ctx context.Context, req dto.SecondaryOTPVerifyRequest, ipAddress, userAgent string) (*dto.AuthResponse, error) {
	sessionUUID, err := uuid.Parse(req.OTPSessionID)
	if err != nil {
		return nil, errors.New("OTP session ID không hợp lệ")
	}

	otpSession, err := s.deviceRepo.FindSecondaryOTPSession(ctx, sessionUUID)
	if err != nil {
		return nil, errors.New("phiên OTP không tồn tại")
	}

	if otpSession.IsUsed {
		return nil, errors.New("mã OTP đã được sử dụng")
	}

	if time.Now().After(otpSession.ExpiresAt) {
		return nil, errors.New("mã OTP đã hết hạn")
	}

	if otpSession.OTPCode != strings.TrimSpace(req.OTPCode) {
		return nil, errors.New("mã OTP không chính xác")
	}

	_ = s.deviceRepo.MarkSecondaryOTPUsed(ctx, sessionUUID)

	account, err := s.authRepo.FindAccountByID(ctx, otpSession.AccountId)
	if err != nil {
		return nil, errors.New("tài khoản không tồn tại")
	}

	return s.completeDeviceLogin(ctx, account, otpSession.DeviceID, otpSession.DeviceName, otpSession.Platform, ipAddress, userAgent)
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
		Platform:     s.Platform,
		IPAddress:    s.IPAddress,
		IsPrimary:    s.IsPrimary,
		IsActive:     s.IsActive,
		IsCurrent:    isCurrent,
		LastActiveAt: s.LastActiveAt.Format(time.RFC3339),
		CreatedAt:    s.CreatedAt.Format(time.RFC3339),
	}
}
