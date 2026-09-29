package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"chatapp/internal/model"
)

type DeviceRepository interface {
	FindSessionByDeviceID(ctx context.Context, accountID uuid.UUID, deviceID string) (*model.DeviceSession, error)
	FindSessionByID(ctx context.Context, id uuid.UUID) (*model.DeviceSession, error)
	CreateSession(ctx context.Context, session *model.DeviceSession) error
	UpdateSession(ctx context.Context, session *model.DeviceSession) error
	ListSessionsByAccountID(ctx context.Context, accountID uuid.UUID) ([]model.DeviceSession, error)
	CountActivePrimarySessions(ctx context.Context, accountID uuid.UUID) (int64, error)
	// CountActivePhoneSessions đếm số phone session đang active (phải luôn ≤ 1)
	CountActivePhoneSessions(ctx context.Context, accountID uuid.UUID) (int64, error)
	// CountActiveSecondarySessions đếm tổng secondary (non-phone) sessions đang active
	CountActiveSecondarySessions(ctx context.Context, accountID uuid.UUID) (int64, error)
	// RevokePhoneSessions thu hồi tất cả phone sessions đang active (dùng khi phone mới login)
	RevokePhoneSessions(ctx context.Context, accountID uuid.UUID) error
	RevokeSession(ctx context.Context, sessionID uuid.UUID) error
	RevokeOtherSessions(ctx context.Context, accountID uuid.UUID, currentSessionID uuid.UUID) error
	SetPrimarySession(ctx context.Context, accountID uuid.UUID, newPrimarySessionID uuid.UUID) error

	// QR Login
	CreateQRSession(ctx context.Context, q *model.QRLoginSession) error
	FindQRSession(ctx context.Context, token string) (*model.QRLoginSession, error)
	ApproveQRSession(ctx context.Context, token string, accountID uuid.UUID, sessionID uuid.UUID) error
	RejectQRSession(ctx context.Context, token string) error

	// Secondary OTP Login
	CreateSecondaryOTPSession(ctx context.Context, s *model.SecondaryOTPSession) error
	FindSecondaryOTPSession(ctx context.Context, id uuid.UUID) (*model.SecondaryOTPSession, error)
	MarkSecondaryOTPUsed(ctx context.Context, id uuid.UUID) error
}

type deviceRepository struct {
	db *gorm.DB
}

func NewDeviceRepository(db *gorm.DB) DeviceRepository {
	return &deviceRepository{db: db}
}

func (r *deviceRepository) FindSessionByDeviceID(ctx context.Context, accountID uuid.UUID, deviceID string) (*model.DeviceSession, error) {
	var session model.DeviceSession
	err := r.db.WithContext(ctx).
		Where("account_id = ? AND device_id = ?", accountID, deviceID).
		First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *deviceRepository) FindSessionByID(ctx context.Context, id uuid.UUID) (*model.DeviceSession, error) {
	var session model.DeviceSession
	err := r.db.WithContext(ctx).First(&session, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *deviceRepository) CreateSession(ctx context.Context, session *model.DeviceSession) error {
	return r.db.WithContext(ctx).Create(session).Error
}

func (r *deviceRepository) UpdateSession(ctx context.Context, session *model.DeviceSession) error {
	return r.db.WithContext(ctx).Save(session).Error
}

func (r *deviceRepository) ListSessionsByAccountID(ctx context.Context, accountID uuid.UUID) ([]model.DeviceSession, error) {
	var sessions []model.DeviceSession
	err := r.db.WithContext(ctx).
		Where("account_id = ? AND is_active = ?", accountID, true).
		Order("is_primary DESC, last_active_at DESC").
		Find(&sessions).Error
	return sessions, err
}

func (r *deviceRepository) CountActivePrimarySessions(ctx context.Context, accountID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.DeviceSession{}).
		Where("account_id = ? AND is_primary = ? AND is_active = ?", accountID, true, true).
		Count(&count).Error
	return count, err
}

func (r *deviceRepository) CountActivePhoneSessions(ctx context.Context, accountID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.DeviceSession{}).
		Where("account_id = ? AND platform = ? AND is_active = ?", accountID, model.PlatformPhone, true).
		Count(&count).Error
	return count, err
}

func (r *deviceRepository) CountActiveSecondarySessions(ctx context.Context, accountID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.DeviceSession{}).
		Where("account_id = ? AND platform != ? AND is_active = ?", accountID, model.PlatformPhone, true).
		Count(&count).Error
	return count, err
}

func (r *deviceRepository) RevokePhoneSessions(ctx context.Context, accountID uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&model.DeviceSession{}).
		Where("account_id = ? AND platform = ? AND is_active = ?", accountID, model.PlatformPhone, true).
		Updates(map[string]interface{}{
			"is_active":  false,
			"is_primary": false,
			"updated_at": time.Now(),
		}).Error
}

func (r *deviceRepository) RevokeSession(ctx context.Context, sessionID uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&model.DeviceSession{}).
		Where("id = ?", sessionID).
		Updates(map[string]interface{}{
			"is_active":  false,
			"updated_at": time.Now(),
		}).Error
}

func (r *deviceRepository) RevokeOtherSessions(ctx context.Context, accountID uuid.UUID, currentSessionID uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&model.DeviceSession{}).
		Where("account_id = ? AND id != ? AND is_active = ?", accountID, currentSessionID, true).
		Updates(map[string]interface{}{
			"is_active":  false,
			"updated_at": time.Now(),
		}).Error
}

func (r *deviceRepository) SetPrimarySession(ctx context.Context, accountID uuid.UUID, newPrimarySessionID uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Hạ tất cả session khác của account xuống is_primary = false
		if err := tx.Model(&model.DeviceSession{}).
			Where("account_id = ?", accountID).
			Update("is_primary", false).Error; err != nil {
			return err
		}

		// Nâng session mới lên is_primary = true và is_active = true
		if err := tx.Model(&model.DeviceSession{}).
			Where("id = ? AND account_id = ?", newPrimarySessionID, accountID).
			Updates(map[string]interface{}{
				"is_primary": true,
				"is_active":  true,
				"updated_at": time.Now(),
			}).Error; err != nil {
			return err
		}

		return nil
	})
}

// ======================== QR Login ========================

func (r *deviceRepository) CreateQRSession(ctx context.Context, q *model.QRLoginSession) error {
	return r.db.WithContext(ctx).Create(q).Error
}

func (r *deviceRepository) FindQRSession(ctx context.Context, token string) (*model.QRLoginSession, error) {
	var q model.QRLoginSession
	err := r.db.WithContext(ctx).First(&q, "token = ?", token).Error
	if err != nil {
		return nil, err
	}
	return &q, nil
}

func (r *deviceRepository) ApproveQRSession(ctx context.Context, token string, accountID uuid.UUID, sessionID uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&model.QRLoginSession{}).
		Where("token = ? AND status = ?", token, "pending").
		Updates(map[string]interface{}{
			"account_id": accountID,
			"session_id": sessionID,
			"status":     "approved",
		}).Error
}

func (r *deviceRepository) RejectQRSession(ctx context.Context, token string) error {
	return r.db.WithContext(ctx).Model(&model.QRLoginSession{}).
		Where("token = ? AND status = ?", token, "pending").
		Update("status", "rejected").Error
}

// ======================== Secondary OTP ========================

func (r *deviceRepository) CreateSecondaryOTPSession(ctx context.Context, s *model.SecondaryOTPSession) error {
	return r.db.WithContext(ctx).Create(s).Error
}

func (r *deviceRepository) FindSecondaryOTPSession(ctx context.Context, id uuid.UUID) (*model.SecondaryOTPSession, error) {
	var s model.SecondaryOTPSession
	err := r.db.WithContext(ctx).First(&s, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *deviceRepository) MarkSecondaryOTPUsed(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&model.SecondaryOTPSession{}).
		Where("id = ?", id).
		Update("is_used", true).Error
}
