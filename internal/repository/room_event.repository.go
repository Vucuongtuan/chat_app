package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"chatapp/internal/model"
)

type RoomEventRepository interface {
	// Pinned Messages
	PinMessage(ctx context.Context, pin *model.PinnedMessage) error
	UnpinMessage(ctx context.Context, roomID, messageID string) error
	FindPinnedMessages(ctx context.Context, roomID string) ([]model.PinnedMessage, error)
	IsMessagePinned(ctx context.Context, roomID, messageID string) (bool, error)

	// Polls
	CreatePoll(ctx context.Context, poll *model.Poll) error
	FindPollByID(ctx context.Context, pollID string) (*model.Poll, error)
	VotePoll(ctx context.Context, pollID string, userUUID uuid.UUID, optionUUIDs []uuid.UUID, isMultiple bool) error

	// Schedules
	CreateSchedule(ctx context.Context, s *model.RoomSchedule) error
	FindSchedulesByRoomID(ctx context.Context, roomID string) ([]model.RoomSchedule, error)
	FindSchedulesByDateRange(ctx context.Context, roomID string, start, end time.Time) ([]model.RoomSchedule, error)
	FindScheduleByID(ctx context.Context, scheduleID string) (*model.RoomSchedule, error)
	DeleteSchedule(ctx context.Context, scheduleID, roomID string) error
}

type roomEventRepository struct {
	db *gorm.DB
}

func NewRoomEventRepository(db *gorm.DB) RoomEventRepository {
	return &roomEventRepository{db: db}
}

// ──────────────────────────────── Pinned Messages ────────────────────────────────

func (r *roomEventRepository) PinMessage(ctx context.Context, pin *model.PinnedMessage) error {
	return r.db.WithContext(ctx).Create(pin).Error
}

func (r *roomEventRepository) UnpinMessage(ctx context.Context, roomID, messageID string) error {
	return r.db.WithContext(ctx).
		Where("room_id = ? AND message_id = ?", roomID, messageID).
		Delete(&model.PinnedMessage{}).Error
}

func (r *roomEventRepository) FindPinnedMessages(ctx context.Context, roomID string) ([]model.PinnedMessage, error) {
	var pins []model.PinnedMessage
	err := r.db.WithContext(ctx).
		Preload("Message").
		Preload("Message.Reactions").
		Where("room_id = ?", roomID).
		Order("pinned_at DESC").
		Find(&pins).Error
	return pins, err
}

func (r *roomEventRepository) IsMessagePinned(ctx context.Context, roomID, messageID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.PinnedMessage{}).
		Where("room_id = ? AND message_id = ?", roomID, messageID).
		Count(&count).Error
	return count > 0, err
}

// ──────────────────────────────── Polls ────────────────────────────────

func (r *roomEventRepository) CreatePoll(ctx context.Context, poll *model.Poll) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(poll).Error; err != nil {
			return err
		}
		return nil
	})
}

func (r *roomEventRepository) FindPollByID(ctx context.Context, pollID string) (*model.Poll, error) {
	var poll model.Poll
	err := r.db.WithContext(ctx).
		Preload("Options.Votes").
		First(&poll, "id = ?", pollID).Error
	if err != nil {
		return nil, err
	}
	return &poll, nil
}

func (r *roomEventRepository) VotePoll(ctx context.Context, pollID string, userUUID uuid.UUID, optionUUIDs []uuid.UUID, isMultiple bool) error {
	pollUUID, err := uuid.Parse(pollID)
	if err != nil {
		return err
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Xóa các vote cũ của user trong poll này
		if err := tx.Where("poll_id = ? AND user_id = ?", pollUUID, userUUID).
			Delete(&model.PollVote{}).Error; err != nil {
			return err
		}

		// Nếu không cho chọn nhiều thì chỉ lấy phương án đầu tiên
		selected := optionUUIDs
		if !isMultiple && len(selected) > 1 {
			selected = selected[:1]
		}

		now := time.Now()
		votes := make([]model.PollVote, 0, len(selected))
		for _, optID := range selected {
			votes = append(votes, model.PollVote{
				ID:        uuid.New(),
				PollId:    pollUUID,
				OptionId:  optID,
				UserId:    userUUID,
				CreatedAt: now,
			})
		}

		if len(votes) > 0 {
			if err := tx.Create(&votes).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// ──────────────────────────────── Room Schedules ────────────────────────────────

func (r *roomEventRepository) CreateSchedule(ctx context.Context, s *model.RoomSchedule) error {
	return r.db.WithContext(ctx).Create(s).Error
}

func (r *roomEventRepository) FindSchedulesByRoomID(ctx context.Context, roomID string) ([]model.RoomSchedule, error) {
	var list []model.RoomSchedule
	err := r.db.WithContext(ctx).
		Where("room_id = ?", roomID).
		Order("event_time ASC").
		Find(&list).Error
	return list, err
}

func (r *roomEventRepository) FindSchedulesByDateRange(ctx context.Context, roomID string, start, end time.Time) ([]model.RoomSchedule, error) {
	var list []model.RoomSchedule
	err := r.db.WithContext(ctx).
		Where("room_id = ? AND event_time >= ? AND event_time <= ?", roomID, start, end).
		Order("event_time ASC").
		Find(&list).Error
	return list, err
}

func (r *roomEventRepository) FindScheduleByID(ctx context.Context, scheduleID string) (*model.RoomSchedule, error) {
	var s model.RoomSchedule
	err := r.db.WithContext(ctx).First(&s, "id = ?", scheduleID).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *roomEventRepository) DeleteSchedule(ctx context.Context, scheduleID, roomID string) error {
	res := r.db.WithContext(ctx).
		Where("id = ? AND room_id = ?", scheduleID, roomID).
		Delete(&model.RoomSchedule{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("không tìm thấy lịch hẹn")
	}
	return nil
}
