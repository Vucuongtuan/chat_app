package service

import (
	"context"
	"fmt"
	"mime/multipart"
	"path/filepath"

	"github.com/google/uuid"

	"chatapp/internal/model"
	"chatapp/internal/repository"
	"chatapp/pkg/storage"
)

type MediaService struct {
	repo    repository.MediaRepository
	storage storage.Storage
}

var allowedExt = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".gif": true,
	".mp4": true, ".mov": true,
}

func NewMediaService(repo repository.MediaRepository, s storage.Storage) *MediaService {
	return &MediaService{repo: repo, storage: s}
}

func (s *MediaService) Upload(ctx context.Context, file *multipart.FileHeader, ownerType string, ownerID uuid.UUID, sortOrder int) (*model.Media, error) {
	if file == nil || file.Size == 0 {
		return nil, fmt.Errorf("file không được để trống")
	}
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
		OwnerId:   ownerID,
		SortOrder: sortOrder,
	}

	if err := s.repo.Create(ctx, media); err != nil {
		_ = s.storage.Delete(filename)
		return nil, err
	}

	return media, nil
}
func (s *MediaService) GetByOwner(ctx context.Context, ownerType string, ownerID uuid.UUID) ([]model.Media, error) {
	return s.repo.FindByOwner(ctx, ownerType, ownerID)
}

func (s *MediaService) Delete(ctx context.Context, id uuid.UUID) error {
	media, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	filename := filepath.Base(media.Url)
	if err := s.storage.Delete(filename); err != nil {
		return err
	}

	return s.repo.Delete(ctx, id)
}
