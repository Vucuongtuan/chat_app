package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"chatapp/internal/model"
)

type AuthRepository interface {
	CreateAccountAndUser(ctx context.Context, account *model.Account, user *model.User, session *model.DeviceSession) error
	FindAccountByID(ctx context.Context, id uuid.UUID) (*model.Account, error)
	FindAccountByPhone(ctx context.Context, phone string) (*model.Account, error)
	FindAccountByMail(ctx context.Context, mail string) (*model.Account, error)
	FindAccountByIdentifier(ctx context.Context, identifier string) (*model.Account, error)
	UpdateAccount(ctx context.Context, account *model.Account) error

	CreateVerificationCode(ctx context.Context, code *model.VerificationCode) error
	FindValidVerificationCode(ctx context.Context, target string, code string, codeType string) (*model.VerificationCode, error)
	MarkVerificationCodeUsed(ctx context.Context, id uuid.UUID) error

	GetUserByAccountID(ctx context.Context, accountID uuid.UUID) (*model.User, error)
}

type authRepository struct {
	db *gorm.DB
}

func NewAuthRepository(db *gorm.DB) AuthRepository {
	return &authRepository{db: db}
}

func (r *authRepository) CreateAccountAndUser(ctx context.Context, account *model.Account, user *model.User, session *model.DeviceSession) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(account).Error; err != nil {
			return err
		}

		user.AccountId = account.ID
		if err := tx.Create(user).Error; err != nil {
			return err
		}

		if session != nil {
			session.AccountId = account.ID
			if err := tx.Create(session).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *authRepository) FindAccountByID(ctx context.Context, id uuid.UUID) (*model.Account, error) {
	var account model.Account
	if err := r.db.WithContext(ctx).Preload("User").First(&account, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func (r *authRepository) FindAccountByPhone(ctx context.Context, phone string) (*model.Account, error) {
	var account model.Account
	if err := r.db.WithContext(ctx).Preload("User").First(&account, "phone = ?", phone).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func (r *authRepository) FindAccountByMail(ctx context.Context, mail string) (*model.Account, error) {
	var account model.Account
	if err := r.db.WithContext(ctx).Preload("User").First(&account, "mail = ?", mail).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func (r *authRepository) FindAccountByIdentifier(ctx context.Context, identifier string) (*model.Account, error) {
	var account model.Account
	err := r.db.WithContext(ctx).Preload("User").
		Where("phone = ? OR mail = ?", identifier, identifier).
		First(&account).Error
	if err != nil {
		return nil, err
	}
	return &account, nil
}

func (r *authRepository) UpdateAccount(ctx context.Context, account *model.Account) error {
	return r.db.WithContext(ctx).Save(account).Error
}

func (r *authRepository) CreateVerificationCode(ctx context.Context, code *model.VerificationCode) error {
	return r.db.WithContext(ctx).Create(code).Error
}

func (r *authRepository) FindValidVerificationCode(ctx context.Context, target string, code string, codeType string) (*model.VerificationCode, error) {
	var vc model.VerificationCode
	now := time.Now()
	err := r.db.WithContext(ctx).
		Where("target = ? AND code = ? AND type = ? AND is_used = ? AND expires_at > ?", target, code, codeType, false, now).
		Order("created_at DESC").
		First(&vc).Error
	if err != nil {
		return nil, err
	}
	return &vc, nil
}

func (r *authRepository) MarkVerificationCodeUsed(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&model.VerificationCode{}).
		Where("id = ?", id).
		Update("is_used", true).Error
}

func (r *authRepository) GetUserByAccountID(ctx context.Context, accountID uuid.UUID) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).First(&user, "account_id = ?", accountID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}
