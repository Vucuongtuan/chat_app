package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"chatapp/internal/model"
)

type RoomRepository interface {
	Create(ctx context.Context, room *model.Room) error
	FindByID(ctx context.Context, id string) (*model.Room, error)
	FindByInviteCode(ctx context.Context, inviteCode string) (*model.Room, error)
	FindDirectRoom(ctx context.Context, user1ID, user2ID string) (*model.Room, error)
	FindByUserID(ctx context.Context, userID string, limit, offset int) ([]model.Room, int64, error)
	Update(ctx context.Context, room *model.Room) error
	Delete(ctx context.Context, id string) error

	AddMember(ctx context.Context, member *model.RoomMember) error
	FindMember(ctx context.Context, roomID, userID string) (*model.RoomMember, error)
	FindMembers(ctx context.Context, roomID string) ([]model.RoomMember, error)
	UpdateMember(ctx context.Context, member *model.RoomMember) error
	RemoveMember(ctx context.Context, roomID, userID string) error
	CountMembers(ctx context.Context, roomID string) (int64, error)
	UpdateLastMessage(ctx context.Context, roomID string, messageID uuid.UUID) error
}

type roomRepository struct {
	db *gorm.DB
}

func NewRoomRepository(db *gorm.DB) RoomRepository {
	return &roomRepository{db: db}
}

func (r *roomRepository) Create(ctx context.Context, room *model.Room) error {
	return r.db.WithContext(ctx).Create(room).Error
}

func (r *roomRepository) FindByID(ctx context.Context, id string) (*model.Room, error) {
	var room model.Room
	err := r.db.WithContext(ctx).
		Preload("Members").
		First(&room, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &room, nil
}

func (r *roomRepository) FindByInviteCode(ctx context.Context, inviteCode string) (*model.Room, error) {
	var room model.Room
	err := r.db.WithContext(ctx).
		Preload("Members").
		First(&room, "invite_code = ?", inviteCode).Error
	if err != nil {
		return nil, err
	}
	return &room, nil
}

func (r *roomRepository) FindDirectRoom(ctx context.Context, user1ID, user2ID string) (*model.Room, error) {
	var room model.Room
	subQuery := r.db.WithContext(ctx).
		Table("room_members").
		Select("room_id").
		Where("user_id IN (?, ?)", user1ID, user2ID).
		Group("room_id").
		Having("COUNT(DISTINCT user_id) = 2")

	err := r.db.WithContext(ctx).
		Preload("Members").
		Where("type = ? AND id IN (?)", "direct", subQuery).
		First(&room).Error
	if err != nil {
		return nil, err
	}
	return &room, nil
}

func (r *roomRepository) FindByUserID(ctx context.Context, userID string, limit, offset int) ([]model.Room, int64, error) {
	var rooms []model.Room
	var total int64

	subQuery := r.db.WithContext(ctx).
		Table("room_members").
		Select("room_id").
		Where("user_id = ?", userID)

	query := r.db.WithContext(ctx).
		Model(&model.Room{}).
		Where("id IN (?)", subQuery)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit <= 0 {
		limit = 20
	}

	err := query.
		Preload("Members").
		Order("updated_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&rooms).Error

	return rooms, total, err
}

func (r *roomRepository) Update(ctx context.Context, room *model.Room) error {
	return r.db.WithContext(ctx).Save(room).Error
}

func (r *roomRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&model.Room{}, "id = ?", id).Error
}

func (r *roomRepository) AddMember(ctx context.Context, member *model.RoomMember) error {
	return r.db.WithContext(ctx).Create(member).Error
}

func (r *roomRepository) FindMember(ctx context.Context, roomID, userID string) (*model.RoomMember, error) {
	var member model.RoomMember
	err := r.db.WithContext(ctx).
		Where("room_id = ? AND user_id = ?", roomID, userID).
		First(&member).Error
	if err != nil {
		return nil, err
	}
	return &member, nil
}

func (r *roomRepository) FindMembers(ctx context.Context, roomID string) ([]model.RoomMember, error) {
	var members []model.RoomMember
	err := r.db.WithContext(ctx).
		Where("room_id = ?", roomID).
		Find(&members).Error
	return members, err
}

func (r *roomRepository) UpdateMember(ctx context.Context, member *model.RoomMember) error {
	return r.db.WithContext(ctx).Save(member).Error
}

func (r *roomRepository) RemoveMember(ctx context.Context, roomID, userID string) error {
	result := r.db.WithContext(ctx).
		Where("room_id = ? AND user_id = ?", roomID, userID).
		Delete(&model.RoomMember{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("member not found")
	}
	return nil
}

func (r *roomRepository) CountMembers(ctx context.Context, roomID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.RoomMember{}).
		Where("room_id = ?", roomID).
		Count(&count).Error
	return count, err
}

func (r *roomRepository) UpdateLastMessage(ctx context.Context, roomID string, messageID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&model.Room{}).
		Where("id = ?", roomID).
		Updates(map[string]interface{}{
			"last_message_id": messageID,
			"updated_at":      time.Now(),
		}).Error
}
