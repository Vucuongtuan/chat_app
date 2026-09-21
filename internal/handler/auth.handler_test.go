package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"chatapp/internal/config"
	"chatapp/internal/dto"
	"chatapp/internal/middleware"
	"chatapp/internal/model"
	"chatapp/internal/service"
	"chatapp/pkg/jwt"
)

// Mock Auth Repository
type mockAuthRepo struct {
	accounts   map[string]*model.Account
	users      map[string]*model.User
	codes      map[string]*model.VerificationCode
	deviceRepo *mockDeviceRepo
}

func newMockAuthRepo(deviceRepo *mockDeviceRepo) *mockAuthRepo {
	return &mockAuthRepo{
		accounts:   make(map[string]*model.Account),
		users:      make(map[string]*model.User),
		codes:      make(map[string]*model.VerificationCode),
		deviceRepo: deviceRepo,
	}
}

func (m *mockAuthRepo) CreateAccountAndUser(ctx context.Context, account *model.Account, user *model.User, session *model.DeviceSession) error {
	if account.ID == uuid.Nil {
		account.ID = uuid.New()
	}
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	user.AccountId = account.ID
	account.User = *user

	m.accounts[account.ID.String()] = account
	m.users[user.ID.String()] = user

	if session != nil && m.deviceRepo != nil {
		if session.ID == uuid.Nil {
			session.ID = uuid.New()
		}
		session.AccountId = account.ID
		_ = m.deviceRepo.CreateSession(ctx, session)
	}

	return nil
}

func (m *mockAuthRepo) FindAccountByID(ctx context.Context, id uuid.UUID) (*model.Account, error) {
	acc, ok := m.accounts[id.String()]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return acc, nil
}

func (m *mockAuthRepo) FindAccountByPhone(ctx context.Context, phone string) (*model.Account, error) {
	for _, acc := range m.accounts {
		if acc.Phone == phone {
			return acc, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockAuthRepo) FindAccountByMail(ctx context.Context, mail string) (*model.Account, error) {
	for _, acc := range m.accounts {
		if acc.Mail != nil && *acc.Mail == mail {
			return acc, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockAuthRepo) FindAccountByIdentifier(ctx context.Context, identifier string) (*model.Account, error) {
	for _, acc := range m.accounts {
		if acc.Phone == identifier || (acc.Mail != nil && *acc.Mail == identifier) {
			return acc, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockAuthRepo) UpdateAccount(ctx context.Context, account *model.Account) error {
	m.accounts[account.ID.String()] = account
	return nil
}

func (m *mockAuthRepo) CreateVerificationCode(ctx context.Context, code *model.VerificationCode) error {
	if code.ID == uuid.Nil {
		code.ID = uuid.New()
	}
	m.codes[code.Code] = code
	return nil
}

func (m *mockAuthRepo) FindValidVerificationCode(ctx context.Context, target string, code string, codeType string) (*model.VerificationCode, error) {
	vc, ok := m.codes[code]
	if !ok || vc.Target != target || vc.Type != codeType || vc.IsUsed || time.Now().After(vc.ExpiresAt) {
		return nil, gorm.ErrRecordNotFound
	}
	return vc, nil
}

func (m *mockAuthRepo) MarkVerificationCodeUsed(ctx context.Context, id uuid.UUID) error {
	for _, vc := range m.codes {
		if vc.ID == id {
			vc.IsUsed = true
			return nil
		}
	}
	return nil
}

func (m *mockAuthRepo) GetUserByAccountID(ctx context.Context, accountID uuid.UUID) (*model.User, error) {
	for _, u := range m.users {
		if u.AccountId == accountID {
			return u, nil
		}
	}
	return nil, nil
}

// Mock Device Repository
type mockDeviceRepo struct {
	sessions map[string]*model.DeviceSession
}

func newMockDeviceRepo() *mockDeviceRepo {
	return &mockDeviceRepo{
		sessions: make(map[string]*model.DeviceSession),
	}
}

func (m *mockDeviceRepo) FindSessionByDeviceID(ctx context.Context, accountID uuid.UUID, deviceID string) (*model.DeviceSession, error) {
	for _, s := range m.sessions {
		if s.AccountId == accountID && s.DeviceID == deviceID {
			return s, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockDeviceRepo) FindSessionByID(ctx context.Context, id uuid.UUID) (*model.DeviceSession, error) {
	s, ok := m.sessions[id.String()]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return s, nil
}

func (m *mockDeviceRepo) CreateSession(ctx context.Context, session *model.DeviceSession) error {
	if session.ID == uuid.Nil {
		session.ID = uuid.New()
	}
	m.sessions[session.ID.String()] = session
	return nil
}

func (m *mockDeviceRepo) UpdateSession(ctx context.Context, session *model.DeviceSession) error {
	m.sessions[session.ID.String()] = session
	return nil
}

func (m *mockDeviceRepo) ListSessionsByAccountID(ctx context.Context, accountID uuid.UUID) ([]model.DeviceSession, error) {
	var list []model.DeviceSession
	for _, s := range m.sessions {
		if s.AccountId == accountID && s.IsActive {
			list = append(list, *s)
		}
	}
	return list, nil
}

func (m *mockDeviceRepo) CountActivePrimarySessions(ctx context.Context, accountID uuid.UUID) (int64, error) {
	var count int64
	for _, s := range m.sessions {
		if s.AccountId == accountID && s.IsPrimary && s.IsActive {
			count++
		}
	}
	return count, nil
}

func (m *mockDeviceRepo) RevokeSession(ctx context.Context, sessionID uuid.UUID) error {
	if s, ok := m.sessions[sessionID.String()]; ok {
		s.IsActive = false
		return nil
	}
	return errors.New("not found")
}

func (m *mockDeviceRepo) RevokeOtherSessions(ctx context.Context, accountID uuid.UUID, currentSessionID uuid.UUID) error {
	for _, s := range m.sessions {
		if s.AccountId == accountID && s.ID != currentSessionID {
			s.IsActive = false
		}
	}
	return nil
}

func (m *mockDeviceRepo) SetPrimarySession(ctx context.Context, accountID uuid.UUID, newPrimarySessionID uuid.UUID) error {
	for _, s := range m.sessions {
		if s.AccountId == accountID {
			s.IsPrimary = false
		}
	}
	if s, ok := m.sessions[newPrimarySessionID.String()]; ok {
		s.IsPrimary = true
		s.IsActive = true
	}
	return nil
}

// Mock Mailer
type mockMailer struct {
	lastOTP string
}

func (m *mockMailer) SendOTP(to string, code string, purpose string) error {
	m.lastOTP = code
	return nil
}

func setupAuthTestRouter() (*gin.Engine, *mockAuthRepo, *mockDeviceRepo, *mockMailer, *config.Config) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		JWTSecret: "test-secret-key-12345",
	}

	deviceRepo := newMockDeviceRepo()
	authRepo := newMockAuthRepo(deviceRepo)
	mailerMock := &mockMailer{}
	authService := service.NewAuthService(authRepo, deviceRepo, mailerMock, cfg)
	authHandler := NewAuthHandler(authService)

	router := gin.Default()
	api := router.Group("/api/v1")
	authMiddleware := middleware.Auth(cfg.JWTSecret)
	authHandler.RegisterRoutes(api, authMiddleware)

	return router, authRepo, deviceRepo, mailerMock, cfg
}

func TestAuth_RegisterAndMultiDevice(t *testing.T) {
	router, _, deviceRepo, _, _ := setupAuthTestRouter()

	// 1. Đăng ký tài khoản
	regBody := dto.RegisterRequest{
		Phone:       "0987654321",
		Mail:        "test@example.com",
		Password:    "password123",
		FullName:    "Test User",
		DisplayName: "testuser",
		DeviceID:    "device-1-primary",
		DeviceName:  "Primary Laptop",
		DeviceType:  "desktop",
	}
	b, _ := json.Marshal(regBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
	}

	var regResp struct {
		Data dto.AuthResponse `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &regResp)

	if !regResp.Data.Session.IsPrimary {
		t.Errorf("expected initial device to be primary")
	}
	primaryToken := regResp.Data.AccessToken
	primarySessionID := regResp.Data.Session.ID

	// 2. Đăng nhập từ thiết bị thứ hai
	loginBody := dto.LoginRequest{
		Identifier: "test@example.com",
		Password:   "password123",
		DeviceID:   "device-2-secondary",
		DeviceName: "Secondary Phone",
		DeviceType: "mobile",
	}
	b2, _ := json.Marshal(loginBody)

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(b2))
	req2.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w2.Code, w2.Body.String())
	}

	var loginResp struct {
		Data dto.AuthResponse `json:"data"`
	}
	_ = json.Unmarshal(w2.Body.Bytes(), &loginResp)

	if loginResp.Data.Session.IsPrimary {
		t.Errorf("expected second device to be secondary (is_primary = false)")
	}
	secondaryToken := loginResp.Data.AccessToken
	secondarySessionID := loginResp.Data.Session.ID

	// 3. Thiết bị phụ cố gắng đăng xuất thiết bị chính -> Phải bị Forbidden 403
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest(http.MethodDelete, "/api/v1/auth/devices/"+primarySessionID, nil)
	req3.Header.Set("Authorization", "Bearer "+secondaryToken)
	router.ServeHTTP(w3, req3)

	if w3.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden when secondary device tries to revoke, got %d: %s", w3.Code, w3.Body.String())
	}

	// 4. Thiết bị chính đăng xuất thiết bị phụ -> Phải thành công 200 OK
	w4 := httptest.NewRecorder()
	req4, _ := http.NewRequest(http.MethodDelete, "/api/v1/auth/devices/"+secondarySessionID, nil)
	req4.Header.Set("Authorization", "Bearer "+primaryToken)
	router.ServeHTTP(w4, req4)

	if w4.Code != http.StatusOK {
		t.Fatalf("expected 200 OK when primary revokes secondary, got %d: %s", w4.Code, w4.Body.String())
	}

	// Kiểm tra session của thiết bị 2 đã bị inactive
	secSession, _ := deviceRepo.FindSessionByID(context.Background(), uuid.MustParse(secondarySessionID))
	if secSession.IsActive {
		t.Errorf("expected secondary session to be inactive")
	}
}

func TestAuth_TwoFactorAuthentication(t *testing.T) {
	router, authRepo, _, mailerMock, _ := setupAuthTestRouter()

	// Khởi tạo sẵn 1 account
	accUUID := uuid.New()
	userUUID := uuid.New()
	hash, _ := bcrypt.GenerateFromPassword([]byte("mypassword"), bcrypt.DefaultCost)
	mailStr := "user2fa@example.com"
	acc := &model.Account{
		ID:               accUUID,
		Phone:            "0911223344",
		Mail:             &mailStr,
		PasswordHash:     string(hash),
		TwoFactorEnabled: false,
	}
	user := &model.User{
		ID:        userUUID,
		AccountId: accUUID,
		FullName:  "User 2FA",
	}
	_ = authRepo.CreateAccountAndUser(context.Background(), acc, user, nil)

	// Tạo token giả lập thiết bị chính
	cfg := &config.Config{JWTSecret: "test-secret-key-12345"}
	primaryToken, _ := jwt.GenerateSessionToken(user, accUUID.String(), "sess-1", "dev-1", true, cfg.JWTSecret, time.Hour)
	secondaryToken, _ := jwt.GenerateSessionToken(user, accUUID.String(), "sess-2", "dev-2", false, cfg.JWTSecret, time.Hour)

	// 1. Thiết bị phụ thử yêu cầu bật 2FA -> Bị 403 Forbidden
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/2fa/enable-request", nil)
	req1.Header.Set("Authorization", "Bearer "+secondaryToken)
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden when non-primary attempts to enable 2FA, got %d", w1.Code)
	}

	// 2. Thiết bị chính yêu cầu bật 2FA -> Thành công và sinh mã OTP
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/2fa/enable-request", nil)
	req2.Header.Set("Authorization", "Bearer "+primaryToken)
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for 2fa enable-request, got %d: %s", w2.Code, w2.Body.String())
	}
	otpCode := mailerMock.lastOTP
	if otpCode == "" {
		t.Fatalf("expected OTP code to be sent")
	}

	// 3. Thiết bị chính xác nhận OTP để kích hoạt 2FA
	confirmBody, _ := json.Marshal(dto.Enable2FAConfirmRequest{OTPCode: otpCode})
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/2fa/enable-confirm", bytes.NewBuffer(confirmBody))
	req3.Header.Set("Authorization", "Bearer "+primaryToken)
	req3.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w3, req3)

	if w3.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for 2fa confirm, got %d: %s", w3.Code, w3.Body.String())
	}

	// Kiểm tra tài khoản đã bật TwoFactorEnabled
	updatedAcc, _ := authRepo.FindAccountByID(context.Background(), accUUID)
	if !updatedAcc.TwoFactorEnabled {
		t.Fatalf("expected account TwoFactorEnabled to be true")
	}

	// 4. Thử đăng nhập lại -> Phải trả về TwoFactorRequired = true
	loginBody, _ := json.Marshal(dto.LoginRequest{
		Identifier: "user2fa@example.com",
		Password:   "mypassword",
		DeviceID:   "dev-3",
	})
	w4 := httptest.NewRecorder()
	req4, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(loginBody))
	req4.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w4, req4)

	var loginResp struct {
		Data dto.AuthResponse `json:"data"`
	}
	_ = json.Unmarshal(w4.Body.Bytes(), &loginResp)

	if !loginResp.Data.TwoFactorRequired {
		t.Fatalf("expected TwoFactorRequired to be true")
	}
	if loginResp.Data.TempToken == "" {
		t.Fatalf("expected TempToken to be returned")
	}

	// 5. Xác thực OTP đăng nhập 2 bước
	loginOTP := mailerMock.lastOTP
	verifyBody, _ := json.Marshal(dto.Verify2FARequest{
		TempToken: loginResp.Data.TempToken,
		OTPCode:   loginOTP,
		DeviceID:  "dev-3",
	})
	w5 := httptest.NewRecorder()
	req5, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/2fa/verify", bytes.NewBuffer(verifyBody))
	req5.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w5, req5)

	if w5.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for 2fa verify, got %d: %s", w5.Code, w5.Body.String())
	}
}
