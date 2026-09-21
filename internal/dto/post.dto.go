package dto

type CreatePostRequest struct {
	Content string `json:"content" binding:"required"`
}

type UpdatePostRequest struct {
	Content string `json:"content" binding:"required"`
}

type CreateCommentRequest struct {
	Content string `json:"content" binding:"required"`
}

type PostCommentResponse struct {
	ID        string `json:"id"`
	PostID    string `json:"post_id"`
	UserID    string `json:"user_id"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type PostLikeResponse struct {
	ID        string `json:"id"`
	PostID    string `json:"post_id"`
	UserID    string `json:"user_id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type PostResponse struct {
	ID            string                `json:"id"`
	Content       string                `json:"content"`
	UserID        string                `json:"user_id"`
	LikesCount    int                   `json:"likes_count"`
	CommentsCount int                   `json:"comments_count"`
	IsLiked       bool                  `json:"is_liked"`
	CreatedAt     string                `json:"created_at"`
	UpdatedAt     string                `json:"updated_at"`
	Comments      []PostCommentResponse `json:"comments,omitempty"`
}
