package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"chatapp/internal/model"
)

type RoomActivityRepository interface {
	CreateActivity(ctx context.Context, act *model.RoomActivity, participants []model.ActivityParticipant) error
	FindActivityByID(ctx context.Context, id string) (*model.RoomActivity, error)
	FindActivitiesByRoomID(ctx context.Context, roomID string, userID string, isRoomAdmin bool) ([]model.RoomActivity, error)
	UpdateActivity(ctx context.Context, act *model.RoomActivity) error
	DeleteActivity(ctx context.Context, id string) error

	FindParticipant(ctx context.Context, activityID, userID string) (*model.ActivityParticipant, error)
	UpdateParticipant(ctx context.Context, p *model.ActivityParticipant) error
	AddParticipant(ctx context.Context, p *model.ActivityParticipant) error
}

type roomActivityRepository struct {
	db *gorm.DB
}

func NewRoomActivityRepository(db *gorm.DB) RoomActivityRepository {
	return &roomActivityRepository{db: db}
}

func (r *roomActivityRepository) CreateActivity(ctx context.Context, act *model.RoomActivity, participants []model.ActivityParticipant) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(act).Error; err != nil {
			return err
		}

		if len(participants) > 0 {
			if err := tx.CreateInBatches(participants, 100).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *roomActivityRepository) FindActivityByID(ctx context.Context, id string) (*model.RoomActivity, error) {
	var act model.RoomActivity
	err := r.db.WithContext(ctx).
		Preload("Participants").
		First(&act, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &act, nil
}

func (r *roomActivityRepository) FindActivitiesByRoomID(ctx context.Context, roomID string, userID string, isRoomAdmin bool) ([]model.RoomActivity, error) {
	var activities []model.RoomActivity

	query := r.db.WithContext(ctx).
		Where("room_id = ?", roomID)

	if !isRoomAdmin {
		// User thường chỉ thấy hoạt động public, hoặc hoạt động mà mình là người tạo hoặc người tham gia
		subQuery := r.db.Model(&model.ActivityParticipant{}).
			Select("activity_id").
			Where("user_id = ?", userID)

		query = query.Where(
			"visibility = ? OR creator_id = ? OR id IN (?)",
			model.VisibilityPublic,
			userID,
			subQuery,
		)
	}

	err := query.
		Preload("Participants").
		Order("created_at DESC").
		Find(&activities).Error

	return activities, err
}

func (r *roomActivityRepository) UpdateActivity(ctx context.Context, act *model.RoomActivity) error {
	return r.db.WithContext(ctx).Save(act).Error
}

func (r *roomActivityRepository) DeleteActivity(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&model.RoomActivity{}, "id = ?", id).Error
}

func (r *roomActivityRepository) FindParticipant(ctx context.Context, activityID, userID string) (*model.ActivityParticipant, error) {
	var p model.ActivityParticipant
	err := r.db.WithContext(ctx).
		Where("activity_id = ? AND user_id = ?", activityID, userID).
		First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *roomActivityRepository) UpdateParticipant(ctx context.Context, p *model.ActivityParticipant) error {
	return r.db.WithContext(ctx).Save(p).Error
}

func (r *roomActivityRepository) AddParticipant(ctx context.Context, p *model.ActivityParticipant) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return r.db.WithContext(ctx).Create(p).Error
}
