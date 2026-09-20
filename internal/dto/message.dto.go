package dto

type SendMessageRequest struct {
	RoomID          string  `json:"room_id" binding:"required"`
	Type            string  `json:"type,omitempty"` // default "text"
	Content         string  `json:"content"`
	ReplyToID       *string `json:"reply_to_id,omitempty"`
	ForwardedFromID *string `json:"forwarded_from_id,omitempty"`
}

type EditMessageRequest struct {
	Content string `json:"content" binding:"required"`
}

type AddReactionRequest struct {
	Emoji string `json:"emoji" binding:"required"`
}

type MessageReactionResponse struct {
	ID        string `json:"id"`
	MessageID string `json:"message_id"`
	UserID    string `json:"user_id"`
	Emoji     string `json:"emoji"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type MessageStatusResponse struct {
	ID          string  `json:"id"`
	MessageID   string  `json:"message_id"`
	UserID      string  `json:"user_id"`
	Status      string  `json:"status"`
	DeliveredAt *string `json:"delivered_at,omitempty"`
	ReadAt      *string `json:"read_at,omitempty"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

type MessageResponse struct {
	ID              string                    `json:"id"`
	RoomID          string                    `json:"room_id"`
	SenderID        string                    `json:"sender_id"`
	Type            string                    `json:"type"`
	Content         string                    `json:"content"`
	ReplyToID       *string                   `json:"reply_to_id,omitempty"`
	ForwardedFromID *string                   `json:"forwarded_from_id,omitempty"`
	IsEdited        bool                      `json:"is_edited"`
	EditedAt        *string                   `json:"edited_at,omitempty"`
	IsDeleted       bool                      `json:"is_deleted"`
	DeletedAt       *string                   `json:"deleted_at,omitempty"`
	CreatedAt       string                    `json:"created_at"`
	UpdatedAt       string                    `json:"updated_at"`
	Reactions       []MessageReactionResponse `json:"reactions,omitempty"`
}
