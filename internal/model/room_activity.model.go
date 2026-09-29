package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Constants cho Room Activity Type
const (
	ActivityTypeTask       = "task"       // Giao việc / Task công việc
	ActivityTypeAttendance = "attendance" // Điểm danh / Check-in buổi học / sự kiện
	ActivityTypeChecklist  = "checklist"  // Danh sách công việc cần tick
	ActivityTypeBilling    = "billing"    // Nhắc học phí / công nợ / quyết toán riêng
	ActivityTypeGeneral    = "general"    // Hoạt động thông báo chung
)

// Constants cho Visibility
const (
	VisibilityPublic   = "public"   // Mọi người trong group đều thấy tiến độ của nhau
	VisibilityTargeted = "targeted" // Chỉ người giao và người được chỉ định mới xem được mục của mình
)

// Constants cho Participant Status
const (
	ParticipantStatusPending   = "pending"   // Đang chờ thực hiện / chưa nộp
	ParticipantStatusSubmitted = "submitted" // Đã hoàn thành / đã nộp bài
	ParticipantStatusConfirmed = "confirmed" // Đã xác nhận / đã duyệt / đã nộp học phí
	ParticipantStatusPresent   = "present"   // Có mặt (điểm danh)
	ParticipantStatusAbsent    = "absent"    // Vắng mặt (điểm danh)
)

// RoomActivity đại diện cho một hoạt động, nhiệm vụ, đợt điểm danh hoặc nhắc học phí trong phòng
type RoomActivity struct {
	ID        uuid.UUID  `json:"id" gorm:"type:char(36);primaryKey"`
	RoomId    uuid.UUID  `json:"room_id" gorm:"type:char(36);not null;index"`
	CreatorId uuid.UUID  `json:"creator_id" gorm:"type:char(36);not null;index"`
	MessageId *uuid.UUID `json:"message_id,omitempty" gorm:"type:char(36);index"` // Liên kết với tin nhắn hiển thị trong phòng

	Type        string `json:"type" gorm:"type:varchar(50);not null;default:'task'"` // task, attendance, checklist, billing
	Title       string `json:"title" gorm:"type:varchar(255);not null"`
	Description string `json:"description" gorm:"type:text"`

	// Visibility: "public" hoặc "targeted" (riêng tư theo từng người)
	Visibility string `json:"visibility" gorm:"type:varchar(20);not null;default:'public'"`

	// Metadata: cấu hình linh hoạt (JSON string) vd: số tiền, tiêu chí, danh sách ngày học tổng quan
	Metadata string `json:"metadata" gorm:"type:text"`

	DueDate   *time.Time `json:"due_date,omitempty"`
	CreatedAt time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time  `json:"updated_at" gorm:"autoUpdateTime"`

	Participants []ActivityParticipant `json:"participants,omitempty" gorm:"foreignKey:ActivityId;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

func (a *RoomActivity) BeforeCreate(tx *gorm.DB) (err error) {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

// ActivityParticipant lưu thông tin trạng thái, tiến độ và ghi chú riêng của từng người trong hoạt động
type ActivityParticipant struct {
	ID         uuid.UUID `json:"id" gorm:"type:char(36);primaryKey"`
	ActivityId uuid.UUID `json:"activity_id" gorm:"type:char(36);not null;index:idx_act_user,unique"`
	UserId     uuid.UUID `json:"user_id" gorm:"type:char(36);not null;index:idx_act_user,unique"`

	// Status: "pending", "submitted", "confirmed", "present", "absent"
	Status string `json:"status" gorm:"type:varchar(50);not null;default:'pending'"`

	// Note: Ghi chú riêng hoặc nội dung tin báo riêng (vd: "Học phí 500.000đ", "Nộp báo cáo qua email")
	Note string `json:"note" gorm:"type:text"`

	// ProgressData: JSON chi tiết (vd: checklist các buổi học [{"date":"2026-09-20","status":"present"}], checkbox con)
	ProgressData string `json:"progress_data" gorm:"type:text"`

	CompletedAt *time.Time `json:"completed_at,omitempty"`
	VerifiedBy  *uuid.UUID `json:"verified_by,omitempty" gorm:"type:char(36)"` // Người duyệt (sếp, giáo viên)

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (p *ActivityParticipant) BeforeCreate(tx *gorm.DB) (err error) {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}
