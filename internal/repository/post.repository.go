package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"chatapp/internal/model"
)

type PostRepository interface {
	Create(ctx context.Context, post *model.Post) error
	FindByID(ctx context.Context, id string) (*model.Post, error)
	List(ctx context.Context, limit, offset int) ([]model.Post, int64, error)
	FindByUserID(ctx context.Context, userID string, limit, offset int) ([]model.Post, int64, error)
	Update(ctx context.Context, post *model.Post) error
	Delete(ctx context.Context, id string) error

	AddLike(ctx context.Context, like *model.PostLike) error
	RemoveLike(ctx context.Context, postID, userID string) error
	IsLiked(ctx context.Context, postID, userID string) (bool, error)
	CountLikes(ctx context.Context, postID string) (int64, error)

	AddComment(ctx context.Context, comment *model.PostComment) error
	FindCommentByID(ctx context.Context, id string) (*model.PostComment, error)
	ListComments(ctx context.Context, postID string, limit, offset int) ([]model.PostComment, int64, error)
	CountComments(ctx context.Context, postID string) (int64, error)
	DeleteComment(ctx context.Context, id string) error
}

type postRepository struct {
	db *gorm.DB
}

func NewPostRepository(db *gorm.DB) PostRepository {
	return &postRepository{db: db}
}

func (r *postRepository) Create(ctx context.Context, post *model.Post) error {
	return r.db.WithContext(ctx).Create(post).Error
}

func (r *postRepository) FindByID(ctx context.Context, id string) (*model.Post, error) {
	var post model.Post
	err := r.db.WithContext(ctx).
		Preload("Likes").
		Preload("Comments").
		First(&post, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &post, nil
}

func (r *postRepository) List(ctx context.Context, limit, offset int) ([]model.Post, int64, error) {
	var posts []model.Post
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Post{})

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit <= 0 {
		limit = 20
	}

	err := query.
		Preload("Likes").
		Preload("Comments").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&posts).Error

	return posts, total, err
}

func (r *postRepository) FindByUserID(ctx context.Context, userID string, limit, offset int) ([]model.Post, int64, error) {
	var posts []model.Post
	var total int64

	query := r.db.WithContext(ctx).
		Model(&model.Post{}).
		Where("user_id = ?", userID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit <= 0 {
		limit = 20
	}

	err := query.
		Preload("Likes").
		Preload("Comments").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&posts).Error

	return posts, total, err
}

func (r *postRepository) Update(ctx context.Context, post *model.Post) error {
	return r.db.WithContext(ctx).Save(post).Error
}

func (r *postRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&model.Post{}, "id = ?", id).Error
}

func (r *postRepository) AddLike(ctx context.Context, like *model.PostLike) error {
	var existing model.PostLike
	err := r.db.WithContext(ctx).
		Where("post_id = ? AND user_id = ?", like.PostId, like.UserId).
		First(&existing).Error
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return r.db.WithContext(ctx).Create(like).Error
}

func (r *postRepository) RemoveLike(ctx context.Context, postID, userID string) error {
	return r.db.WithContext(ctx).
		Where("post_id = ? AND user_id = ?", postID, userID).
		Delete(&model.PostLike{}).Error
}

func (r *postRepository) IsLiked(ctx context.Context, postID, userID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.PostLike{}).
		Where("post_id = ? AND user_id = ?", postID, userID).
		Count(&count).Error
	return count > 0, err
}

func (r *postRepository) CountLikes(ctx context.Context, postID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.PostLike{}).
		Where("post_id = ?", postID).
		Count(&count).Error
	return count, err
}

func (r *postRepository) AddComment(ctx context.Context, comment *model.PostComment) error {
	return r.db.WithContext(ctx).Create(comment).Error
}

func (r *postRepository) FindCommentByID(ctx context.Context, id string) (*model.PostComment, error) {
	var comment model.PostComment
	err := r.db.WithContext(ctx).
		First(&comment, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &comment, nil
}

func (r *postRepository) ListComments(ctx context.Context, postID string, limit, offset int) ([]model.PostComment, int64, error) {
	var comments []model.PostComment
	var total int64

	query := r.db.WithContext(ctx).
		Model(&model.PostComment{}).
		Where("post_id = ?", postID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit <= 0 {
		limit = 20
	}

	err := query.
		Order("created_at ASC").
		Limit(limit).
		Offset(offset).
		Find(&comments).Error

	return comments, total, err
}

func (r *postRepository) CountComments(ctx context.Context, postID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.PostComment{}).
		Where("post_id = ?", postID).
		Count(&count).Error
	return count, err
}

func (r *postRepository) DeleteComment(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&model.PostComment{}, "id = ?", id).Error
}
