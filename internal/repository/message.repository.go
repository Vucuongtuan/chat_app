package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"chatapp/internal/model"
)

type MessageRepository interface {
	Create(ctx context.Context, msg *model.Message, recipientIDs []uuid.UUID) error
	FindByID(ctx context.Context, id string) (*model.Message, error)
	FindByRoomID(ctx context.Context, roomID string, limit, offset int, beforeID *string) ([]model.Message, int64, error)
	Update(ctx context.Context, msg *model.Message) error
	SoftDelete(ctx context.Context, id string) error

	AddReaction(ctx context.Context, reaction *model.MessageReaction) error
	RemoveReaction(ctx context.Context, messageID, userID, emoji string) error
	FindReactions(ctx context.Context, messageID string) ([]model.MessageReaction, error)

	UpdateStatus(ctx context.Context, messageID, userID, status string) error
	MarkRoomMessagesAsRead(ctx context.Context, roomID, userID string, lastReadMessageID string) error
}

type messageRepository struct {
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) MessageRepository {
	return &messageRepository{db: db}
}

func (r *messageRepository) Create(ctx context.Context, msg *model.Message, recipientIDs []uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(msg).Error; err != nil {
			return err
		}

		now := time.Now()
		statuses := make([]model.MessageStatus, 0, len(recipientIDs))
		for _, recipientID := range recipientIDs {
			if recipientID == msg.SenderId {
				continue
			}
			statuses = append(statuses, model.MessageStatus{
				MessageId: msg.ID,
				UserId:    recipientID,
				Status:    "sent",
				CreatedAt: now,
				UpdatedAt: now,
			})
		}
		if len(statuses) > 0 {
			if err := tx.CreateInBatches(statuses, 100).Error; err != nil {
				return err
			}
		}

		return tx.Model(&model.Room{}).
			Where("id = ?", msg.RoomId).
			Updates(map[string]interface{}{
				"last_message_id": msg.ID,
				"updated_at":      now,
			}).Error
	})
}

func (r *messageRepository) FindByID(ctx context.Context, id string) (*model.Message, error) {
	var msg model.Message
	err := r.db.WithContext(ctx).
		Preload("Reactions").
		First(&msg, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

func (r *messageRepository) FindByRoomID(ctx context.Context, roomID string, limit, offset int, beforeID *string) ([]model.Message, int64, error) {
	var messages []model.Message
	var total int64

	query := r.db.WithContext(ctx).
		Model(&model.Message{}).
		Where("room_id = ?", roomID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if beforeID != nil && *beforeID != "" {
		var beforeMsg model.Message
		if err := r.db.WithContext(ctx).Select("created_at").First(&beforeMsg, "id = ?", *beforeID).Error; err == nil {
			query = query.Where("created_at < ?", beforeMsg.CreatedAt)
		}
	}

	if limit <= 0 {
		limit = 30
	}

	err := query.
		Preload("Reactions").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&messages).Error

	return messages, total, err
}

func (r *messageRepository) Update(ctx context.Context, msg *model.Message) error {
	return r.db.WithContext(ctx).Save(msg).Error
}

func (r *messageRepository) SoftDelete(ctx context.Context, id string) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.Message{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_deleted": true,
			"deleted_at": &now,
			"content":    "",
		}).Error
}

func (r *messageRepository) AddReaction(ctx context.Context, reaction *model.MessageReaction) error {
	var existing model.MessageReaction
	err := r.db.WithContext(ctx).
		Where("message_id = ? AND user_id = ? AND emoji = ?", reaction.MessageId, reaction.UserId, reaction.Emoji).
		First(&existing).Error

	if err == nil {
		return nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	return r.db.WithContext(ctx).Create(reaction).Error
}

func (r *messageRepository) RemoveReaction(ctx context.Context, messageID, userID, emoji string) error {
	return r.db.WithContext(ctx).
		Where("message_id = ? AND user_id = ? AND emoji = ?", messageID, userID, emoji).
		Delete(&model.MessageReaction{}).Error
}

func (r *messageRepository) FindReactions(ctx context.Context, messageID string) ([]model.MessageReaction, error) {
	var reactions []model.MessageReaction
	err := r.db.WithContext(ctx).
		Where("message_id = ?", messageID).
		Find(&reactions).Error
	return reactions, err
}

func (r *messageRepository) UpdateStatus(ctx context.Context, messageID, userID, status string) error {
	now := time.Now()
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": now,
	}
	if status == "delivered" {
		updates["delivered_at"] = &now
	} else if status == "read" {
		updates["read_at"] = &now
	}

	return r.db.WithContext(ctx).
		Model(&model.MessageStatus{}).
		Where("message_id = ? AND user_id = ?", messageID, userID).
		Updates(updates).Error
}

func (r *messageRepository) MarkRoomMessagesAsRead(ctx context.Context, roomID, userID string, lastReadMessageID string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Update statuses
		subQuery := tx.Model(&model.Message{}).
			Select("id").
			Where("room_id = ?", roomID)

		if err := tx.Model(&model.MessageStatus{}).
			Where("user_id = ? AND status != ? AND message_id IN (?)", userID, "read", subQuery).
			Updates(map[string]interface{}{
				"status":     "read",
				"read_at":    &now,
				"updated_at": now,
			}).Error; err != nil {
			return err
		}

		// Update member last read message
		if lastReadMessageID != "" {
			parsedID, err := uuid.Parse(lastReadMessageID)
			if err == nil {
				if err := tx.Model(&model.RoomMember{}).
					Where("room_id = ? AND user_id = ?", roomID, userID).
					Updates(map[string]interface{}{
						"last_read_message_id": &parsedID,
						"last_read_at":         &now,
					}).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})
}
