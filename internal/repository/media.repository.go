package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"chatapp/internal/model"
)

// MediaRepository contains persistence operations for media records.
type MediaRepository interface {
	Create(ctx context.Context, media *model.Media) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.Media, error)
	FindByOwner(ctx context.Context, ownerType string, ownerID uuid.UUID) ([]model.Media, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type mediaRepository struct{ db *gorm.DB }

func NewMediaRepository(db *gorm.DB) MediaRepository { return &mediaRepository{db: db} }

func (r *mediaRepository) Create(ctx context.Context, media *model.Media) error {
	return r.db.WithContext(ctx).Create(media).Error
}

func (r *mediaRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Media, error) {
	var media model.Media
	if err := r.db.WithContext(ctx).First(&media, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &media, nil
}

func (r *mediaRepository) FindByOwner(ctx context.Context, ownerType string, ownerID uuid.UUID) ([]model.Media, error) {
	var media []model.Media
	err := r.db.WithContext(ctx).Where("owner_type = ? AND owner_id = ?", ownerType, ownerID).
		Order("sort_order ASC, created_at ASC").Find(&media).Error
	return media, err
}

func (r *mediaRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.Media{}, "id = ?", id).Error
}
