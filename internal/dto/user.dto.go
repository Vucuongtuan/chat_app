package dto

type CreateUserRequest struct {
	Phone       string
	Mail        string
	Password    string
	FullName    string
	DisplayName string
}

type CreateUserResponse struct {
	UserID string `json:"user_id"`
}

type UpdateProfileRequest struct {
	FullName    *string `json:"full_name"`
	DisplayName *string `json:"display_name"`
	AvatarUrl   *string `json:"avatar_url"`
	CoverUrl    *string `json:"cover_url"`
	Bio         *string `json:"bio"`
	Status      *string `json:"status"`
}

type UserResponse struct {
	ID          string  `json:"id"`
	FullName    string  `json:"full_name"`
	DisplayName string  `json:"display_name"`
	AvatarUrl   string  `json:"avatar_url"`
	CoverUrl    string  `json:"cover_url"`
	Bio         string  `json:"bio"`
	Status      string  `json:"status"`
	LastLoginAt *string `json:"last_login_at,omitempty"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}
