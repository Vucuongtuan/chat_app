package service

import (
	"fmt"
	"mime/multipart"
	"path/filepath"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"chatapp/internal/model"
	"chatapp/pkg/storage"
)

type MediaService struct {
	db      *gorm.DB
	storage storage.Storage
}

var allowedExt = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".gif": true,
	".mp4": true, ".mov": true,
}

func NewMediaService(db *gorm.DB, s storage.Storage) *MediaService {
	return &MediaService{db: db, storage: s}
}

func (s *MediaService) Upload(file *multipart.FileHeader, ownerType string, ownerId uint, sortOrder int) (*model.Media, error) {
	ext := filepath.Ext(file.Filename)
	if !allowedExt[ext] {
		return nil, fmt.Errorf("định dạng file không được hỗ trợ: %s", ext)
	}

	mediaType := "image"
	if ext == ".mp4" || ext == ".mov" {
		mediaType = "video"
	}

	filename := fmt.Sprintf("%s%s", uuid.New().String(), ext)

	url, err := s.storage.Upload(file, filename) // gọi qua interface, không biết local hay R2
	if err != nil {
		return nil, err
	}

	media := &model.Media{
		Url:       url,
		Type:      mediaType,
		OwnerType: ownerType,
		OwnerId:   ownerId,
		SortOrder: sortOrder,
	}

	if err := s.db.Create(media).Error; err != nil {
		return nil, err
	}

	return media, nil
}
func (s *MediaService) GetByOwner(ownerType string, ownerId uint) ([]model.Media, error) {
	var media []model.Media
	err := s.db.Where("owner_type = ? AND owner_id = ?", ownerType, ownerId).
		Order("sort_order asc").
		Find(&media).Error
	return media, err
}

func (s *MediaService) Delete(id uint) error {
	var media model.Media
	if err := s.db.First(&media, id).Error; err != nil {
		return err
	}

	filename := filepath.Base(media.Url)
	if err := s.storage.Delete(filename); err != nil {
		return err
	}

	return s.db.Delete(&model.Media{}, id).Error
}
