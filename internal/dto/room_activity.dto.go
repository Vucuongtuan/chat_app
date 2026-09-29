package dto

type ActivityParticipantInput struct {
	UserID       string `json:"user_id" binding:"required"`
	Status       string `json:"status,omitempty"`        // pending, present, absent, etc.
	Note         string `json:"note,omitempty"`          // Lời nhắn riêng (vd: học phí 500k, task yêu cầu)
	ProgressData string `json:"progress_data,omitempty"` // JSON string cấu hình chi tiết / checklist
}

type CreateActivityRequest struct {
	Type         string                     `json:"type" binding:"required,oneof=task attendance checklist billing general"`
	Title        string                     `json:"title" binding:"required"`
	Description  string                     `json:"description,omitempty"`
	Visibility   string                     `json:"visibility" binding:"required,oneof=public targeted"`
	Metadata     string                     `json:"metadata,omitempty"` // JSON string cấu hình
	DueDate      *string                    `json:"due_date,omitempty"` // RFC3339
	Participants []ActivityParticipantInput `json:"participants,omitempty"`
}

type UpdateParticipantRequest struct {
	Status       *string `json:"status,omitempty"`
	Note         *string `json:"note,omitempty"`
	ProgressData *string `json:"progress_data,omitempty"`
	IsVerified   *bool   `json:"is_verified,omitempty"` // Sếp / giáo viên xác nhận
}

type ActivityParticipantResponse struct {
	ID           string  `json:"id"`
	ActivityID   string  `json:"activity_id"`
	UserID       string  `json:"user_id"`
	Status       string  `json:"status"`
	Note         string  `json:"note"`
	ProgressData string  `json:"progress_data"`
	CompletedAt  *string `json:"completed_at,omitempty"`
	VerifiedBy   *string `json:"verified_by,omitempty"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

type ActivityResponse struct {
	ID           string                        `json:"id"`
	RoomID       string                        `json:"room_id"`
	CreatorID    string                        `json:"creator_id"`
	MessageID    *string                       `json:"message_id,omitempty"`
	Type         string                        `json:"type"`
	Title        string                        `json:"title"`
	Description  string                        `json:"description"`
	Visibility   string                        `json:"visibility"`
	Metadata     string                        `json:"metadata"`
	DueDate      *string                       `json:"due_date,omitempty"`
	CreatedAt    string                        `json:"created_at"`
	UpdatedAt    string                        `json:"updated_at"`
	Participants []ActivityParticipantResponse `json:"participants,omitempty"`
}
