package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"chatapp/internal/dto"
	"chatapp/internal/model"
	"chatapp/internal/repository"
)

type PostService struct {
	postRepo repository.PostRepository
	userRepo repository.UserRepository
}

func NewPostService(postRepo repository.PostRepository, userRepo repository.UserRepository) *PostService {
	return &PostService{
		postRepo: postRepo,
		userRepo: userRepo,
	}
}

func (s *PostService) CreatePost(ctx context.Context, userIDStr string, req dto.CreatePostRequest) (*model.Post, error) {
	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("user id không hợp lệ")
	}

	content := strings.TrimSpace(req.Content)
	if content == "" {
		return nil, fmt.Errorf("nội dung bài viết không được để trống")
	}

	now := time.Now()
	post := &model.Post{
		ID:        uuid.New(),
		UserId:    userUUID,
		Content:   content,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.postRepo.Create(ctx, post); err != nil {
		return nil, err
	}

	return post, nil
}

func (s *PostService) GetPost(ctx context.Context, currentUserID, postID string) (*dto.PostResponse, error) {
	post, err := s.postRepo.FindByID(ctx, postID)
	if err != nil {
		return nil, err
	}

	isLiked := false
	if currentUserID != "" {
		isLiked, _ = s.postRepo.IsLiked(ctx, postID, currentUserID)
	}

	res := s.toPostResponse(post, isLiked)
	return &res, nil
}

func (s *PostService) ListPosts(ctx context.Context, currentUserID string, page, perPage int) ([]dto.PostResponse, int64, error) {
	if page < 1 {
		page = 1
	}
	if perPage <= 0 || perPage > 50 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	posts, total, err := s.postRepo.List(ctx, perPage, offset)
	if err != nil {
		return nil, 0, err
	}

	result := make([]dto.PostResponse, len(posts))
	for i, p := range posts {
		isLiked := false
		if currentUserID != "" {
			isLiked, _ = s.postRepo.IsLiked(ctx, p.ID.String(), currentUserID)
		}
		result[i] = s.toPostResponse(&p, isLiked)
	}

	return result, total, nil
}

func (s *PostService) ListUserPosts(ctx context.Context, currentUserID, targetUserID string, page, perPage int) ([]dto.PostResponse, int64, error) {
	if page < 1 {
		page = 1
	}
	if perPage <= 0 || perPage > 50 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	posts, total, err := s.postRepo.FindByUserID(ctx, targetUserID, perPage, offset)
	if err != nil {
		return nil, 0, err
	}

	result := make([]dto.PostResponse, len(posts))
	for i, p := range posts {
		isLiked := false
		if currentUserID != "" {
			isLiked, _ = s.postRepo.IsLiked(ctx, p.ID.String(), currentUserID)
		}
		result[i] = s.toPostResponse(&p, isLiked)
	}

	return result, total, nil
}

func (s *PostService) UpdatePost(ctx context.Context, userID, postID string, req dto.UpdatePostRequest) (*model.Post, error) {
	post, err := s.postRepo.FindByID(ctx, postID)
	if err != nil {
		return nil, err
	}

	if post.UserId.String() != userID {
		return nil, fmt.Errorf("bạn chỉ có thể chỉnh sửa bài viết của chính mình")
	}

	content := strings.TrimSpace(req.Content)
	if content == "" {
		return nil, fmt.Errorf("nội dung bài viết không được để trống")
	}

	post.Content = content
	post.UpdatedAt = time.Now()

	if err := s.postRepo.Update(ctx, post); err != nil {
		return nil, err
	}

	return post, nil
}

func (s *PostService) DeletePost(ctx context.Context, userID, postID string) error {
	post, err := s.postRepo.FindByID(ctx, postID)
	if err != nil {
		return err
	}

	if post.UserId.String() != userID {
		return fmt.Errorf("bạn không có quyền xóa bài viết này")
	}

	return s.postRepo.Delete(ctx, postID)
}

func (s *PostService) ToggleLike(ctx context.Context, userIDStr, postIDStr string) (bool, error) {
	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		return false, fmt.Errorf("user id không hợp lệ")
	}
	postUUID, err := uuid.Parse(postIDStr)
	if err != nil {
		return false, fmt.Errorf("post id không hợp lệ")
	}

	// Verify post exists
	if _, err := s.postRepo.FindByID(ctx, postIDStr); err != nil {
		return false, err
	}

	isLiked, err := s.postRepo.IsLiked(ctx, postIDStr, userIDStr)
	if err != nil {
		return false, err
	}

	if isLiked {
		if err := s.postRepo.RemoveLike(ctx, postIDStr, userIDStr); err != nil {
			return false, err
		}
		return false, nil
	}

	now := time.Now()
	like := &model.PostLike{
		ID:        uuid.New(),
		PostId:    postUUID,
		UserId:    userUUID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.postRepo.AddLike(ctx, like); err != nil {
		return false, err
	}

	return true, nil
}

func (s *PostService) AddComment(ctx context.Context, userIDStr, postIDStr string, req dto.CreateCommentRequest) (*model.PostComment, error) {
	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("user id không hợp lệ")
	}
	postUUID, err := uuid.Parse(postIDStr)
	if err != nil {
		return nil, fmt.Errorf("post id không hợp lệ")
	}

	// Verify post exists
	if _, err := s.postRepo.FindByID(ctx, postIDStr); err != nil {
		return nil, err
	}

	content := strings.TrimSpace(req.Content)
	if content == "" {
		return nil, fmt.Errorf("nội dung bình luận không được để trống")
	}

	now := time.Now()
	comment := &model.PostComment{
		ID:        uuid.New(),
		PostId:    postUUID,
		UserId:    userUUID,
		Content:   content,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.postRepo.AddComment(ctx, comment); err != nil {
		return nil, err
	}

	return comment, nil
}

func (s *PostService) ListComments(ctx context.Context, postID string, page, perPage int) ([]model.PostComment, int64, error) {
	if page < 1 {
		page = 1
	}
	if perPage <= 0 || perPage > 50 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	return s.postRepo.ListComments(ctx, postID, perPage, offset)
}

func (s *PostService) DeleteComment(ctx context.Context, userID, commentID string) error {
	comment, err := s.postRepo.FindCommentByID(ctx, commentID)
	if err != nil {
		return err
	}

	if comment.UserId.String() != userID {
		// Also allow post author to delete comments on their post
		post, err := s.postRepo.FindByID(ctx, comment.PostId.String())
		if err != nil || post.UserId.String() != userID {
			return fmt.Errorf("bạn không có quyền xóa bình luận này")
		}
	}

	return s.postRepo.DeleteComment(ctx, commentID)
}

func (s *PostService) toPostResponse(post *model.Post, isLiked bool) dto.PostResponse {
	comments := make([]dto.PostCommentResponse, len(post.Comments))
	for i, c := range post.Comments {
		comments[i] = dto.PostCommentResponse{
			ID:        c.ID.String(),
			PostID:    c.PostId.String(),
			UserID:    c.UserId.String(),
			Content:   c.Content,
			CreatedAt: c.CreatedAt.Format(time.RFC3339),
			UpdatedAt: c.UpdatedAt.Format(time.RFC3339),
		}
	}

	return dto.PostResponse{
		ID:            post.ID.String(),
		Content:       post.Content,
		UserID:        post.UserId.String(),
		LikesCount:    len(post.Likes),
		CommentsCount: len(post.Comments),
		IsLiked:       isLiked,
		CreatedAt:     post.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     post.UpdatedAt.Format(time.RFC3339),
		Comments:      comments,
	}
}
