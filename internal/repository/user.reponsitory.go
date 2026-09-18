package repository

import (
	"context"

	"chatapp/internal/model"
	"gorm.io/gorm"
)

type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	CreateUser(ctx context.Context, account *model.Account, user *model.User) error
	AccountExistsByPhone(ctx context.Context, phone string) (bool, error)
	AccountExistsByMail(ctx context.Context, mail string) (bool, error)

	FindByID(ctx context.Context, id string) (*model.User, error)
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindByPhone(ctx context.Context, phone string) (*model.User, error)
	FindByDisplayName(ctx context.Context, displayName string) (*model.User, error)
	Search(ctx context.Context, query string, limit int) ([]model.User, error)

	Update(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, id string) error
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *userRepository) CreateUser(ctx context.Context, account *model.Account, user *model.User) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(account).Error; err != nil {
			return err
		}
		user.AccountId = account.ID
		return tx.Create(user).Error
	})
}

func (r *userRepository) AccountExistsByPhone(ctx context.Context, phone string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Account{}).Where("phone = ?", phone).Count(&count).Error
	return count > 0, err
}

func (r *userRepository) AccountExistsByMail(ctx context.Context, mail string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Account{}).Where("mail = ?", mail).Count(&count).Error
	return count > 0, err
}

func (r *userRepository) FindByID(ctx context.Context, id string) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).First(&user, "email = ?", email).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) FindByPhone(ctx context.Context, phone string) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).First(&user, "phone = ?", phone).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) FindByDisplayName(ctx context.Context, displayName string) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).First(&user, "display_name = ?", displayName).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) Search(ctx context.Context, query string, limit int) ([]model.User, error) {
	var users []model.User
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	q := "%" + query + "%"
	err := r.db.WithContext(ctx).
		Where("display_name LIKE ? OR full_name LIKE ?", q, q).
		Limit(limit).
		Find(&users).Error
	return users, err
}

func (r *userRepository) Update(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *userRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&model.User{}, "id = ?", id).Error
}
