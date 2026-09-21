package dto

type CreateRoomRequest struct {
	Type        string   `json:"type" binding:"required,oneof=direct group"`
	Name        *string  `json:"name,omitempty"`
	Description *string  `json:"description,omitempty"`
	AvatarUrl   *string  `json:"avatar_url,omitempty"`
	MemberIDs   []string `json:"member_ids,omitempty"`
}

type UpdateRoomRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	AvatarUrl   *string `json:"avatar_url,omitempty"`
}

type AddRoomMemberRequest struct {
	UserID   string  `json:"user_id" binding:"required"`
	Role     *string `json:"role,omitempty"`
	Nickname *string `json:"nickname,omitempty"`
}

type UpdateRoomMemberRequest struct {
	Role       *string `json:"role,omitempty"`
	Nickname   *string `json:"nickname,omitempty"`
	IsMuted    *bool   `json:"is_muted,omitempty"`
	IsArchived *bool   `json:"is_archived,omitempty"`
	IsPinned   *bool   `json:"is_pinned,omitempty"`
}

type JoinRoomByInviteCodeRequest struct {
	InviteCode string `json:"invite_code" binding:"required"`
}

type RoomMemberResponse struct {
	ID                string  `json:"id"`
	RoomID            string  `json:"room_id"`
	UserID            string  `json:"user_id"`
	Role              string  `json:"role"`
	Nickname          *string `json:"nickname,omitempty"`
	JoinedAt          string  `json:"joined_at"`
	IsMuted           bool    `json:"is_muted"`
	IsArchived        bool    `json:"is_archived"`
	IsPinned          bool    `json:"is_pinned"`
	LastReadMessageID *string `json:"last_read_message_id,omitempty"`
	LastReadAt        *string `json:"last_read_at,omitempty"`
	CreatedAt         string  `json:"created_at"`
	UpdatedAt         string  `json:"updated_at"`
}

type RoomResponse struct {
	ID            string               `json:"id"`
	Type          string               `json:"type"`
	Name          *string              `json:"name,omitempty"`
	Description   *string              `json:"description,omitempty"`
	AvatarUrl     *string              `json:"avatar_url,omitempty"`
	CreatedBy     string               `json:"created_by"`
	InviteCode    *string              `json:"invite_code,omitempty"`
	LastMessageID *string              `json:"last_message_id,omitempty"`
	CreatedAt     string               `json:"created_at"`
	UpdatedAt     string               `json:"updated_at"`
	Members       []RoomMemberResponse `json:"members,omitempty"`
}
