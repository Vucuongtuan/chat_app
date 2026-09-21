package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"chatapp/internal/dto"
	"chatapp/internal/model"
	"chatapp/internal/repository"
	"chatapp/pkg/eventbus"
)

type MessageService struct {
	messageRepo repository.MessageRepository
	roomRepo    repository.RoomRepository
	bus         *eventbus.Bus
}

func NewMessageService(messageRepo repository.MessageRepository, roomRepo repository.RoomRepository, bus *eventbus.Bus) *MessageService {
	return &MessageService{
		messageRepo: messageRepo,
		roomRepo:    roomRepo,
		bus:         bus,
	}
}

func (s *MessageService) SendMessage(ctx context.Context, senderIDStr string, req dto.SendMessageRequest) (*model.Message, error) {
	senderID, err := uuid.Parse(senderIDStr)
	if err != nil {
		return nil, fmt.Errorf("sender id không hợp lệ")
	}

	roomIDStr := strings.TrimSpace(req.RoomID)
	roomID, err := uuid.Parse(roomIDStr)
	if err != nil {
		return nil, fmt.Errorf("room id không hợp lệ")
	}

	// Verify sender is a member of the room
	_, err = s.roomRepo.FindMember(ctx, roomIDStr, senderIDStr)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("bạn không phải thành viên của phòng này")
		}
		return nil, err
	}

	msgType := strings.ToLower(strings.TrimSpace(req.Type))
	if msgType == "" {
		msgType = "text"
	}

	content := strings.TrimSpace(req.Content)
	if msgType == "text" && content == "" {
		return nil, fmt.Errorf("nội dung tin nhắn không được để trống")
	}

	var replyToUUID *uuid.UUID
	if req.ReplyToID != nil && strings.TrimSpace(*req.ReplyToID) != "" {
		parsed, err := uuid.Parse(strings.TrimSpace(*req.ReplyToID))
		if err != nil {
			return nil, fmt.Errorf("reply_to_id không hợp lệ")
		}
		replyMsg, err := s.messageRepo.FindByID(ctx, parsed.String())
		if err != nil || replyMsg.RoomId != roomID {
			return nil, fmt.Errorf("tin nhắn trả lời không tồn tại trong phòng này")
		}
		replyToUUID = &parsed
	}

	var forwardUUID *uuid.UUID
	if req.ForwardedFromID != nil && strings.TrimSpace(*req.ForwardedFromID) != "" {
		parsed, err := uuid.Parse(strings.TrimSpace(*req.ForwardedFromID))
		if err != nil {
			return nil, fmt.Errorf("forwarded_from_id không hợp lệ")
		}
		forwardUUID = &parsed
	}

	// Get all room members to create status
	members, err := s.roomRepo.FindMembers(ctx, roomIDStr)
	if err != nil {
		return nil, err
	}
	var recipientIDs []uuid.UUID
	for _, m := range members {
		if m.UserId != senderID {
			recipientIDs = append(recipientIDs, m.UserId)
		}
	}

	now := time.Now()
	msg := &model.Message{
		ID:              uuid.New(),
		RoomId:          roomID,
		SenderId:        senderID,
		Type:            msgType,
		Content:         content,
		ReplyToId:       replyToUUID,
		ForwardedFromId: forwardUUID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.messageRepo.Create(ctx, msg, recipientIDs); err != nil {
		return nil, err
	}

	return msg, nil
}

func (s *MessageService) GetMessage(ctx context.Context, userID, messageID string) (*model.Message, error) {
	msg, err := s.messageRepo.FindByID(ctx, messageID)
	if err != nil {
		return nil, err
	}

	// Check user is room member
	_, err = s.roomRepo.FindMember(ctx, msg.RoomId.String(), userID)
	if err != nil {
		return nil, fmt.Errorf("bạn không phải thành viên của phòng chứa tin nhắn này")
	}

	return msg, nil
}

func (s *MessageService) ListMessages(ctx context.Context, userID, roomID string, page, perPage int, beforeID *string) ([]model.Message, int64, error) {
	// Check user is room member
	_, err := s.roomRepo.FindMember(ctx, roomID, userID)
	if err != nil {
		return nil, 0, fmt.Errorf("bạn không phải thành viên của phòng này")
	}

	if perPage <= 0 || perPage > 100 {
		perPage = 30
	}
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * perPage

	return s.messageRepo.FindByRoomID(ctx, roomID, perPage, offset, beforeID)
}

func (s *MessageService) EditMessage(ctx context.Context, userID, messageID string, req dto.EditMessageRequest) (*model.Message, error) {
	msg, err := s.messageRepo.FindByID(ctx, messageID)
	if err != nil {
		return nil, err
	}

	if msg.SenderId.String() != userID {
		return nil, fmt.Errorf("bạn chỉ có thể chỉnh sửa tin nhắn của chính mình")
	}

	if msg.IsDeleted {
		return nil, fmt.Errorf("tin nhắn đã bị xóa")
	}

	content := strings.TrimSpace(req.Content)
	if content == "" {
		return nil, fmt.Errorf("nội dung tin nhắn không được để trống")
	}

	now := time.Now()
	msg.Content = content
	msg.IsEdited = true
	msg.EditedAt = &now
	msg.UpdatedAt = now

	if err := s.messageRepo.Update(ctx, msg); err != nil {
		return nil, err
	}

	return msg, nil
}

func (s *MessageService) DeleteMessage(ctx context.Context, userID, messageID string) error {
	msg, err := s.messageRepo.FindByID(ctx, messageID)
	if err != nil {
		return err
	}

	if msg.SenderId.String() != userID {
		// If not sender, check if user is room owner/admin
		member, err := s.roomRepo.FindMember(ctx, msg.RoomId.String(), userID)
		if err != nil || (member.Role != "owner" && member.Role != "admin") {
			return fmt.Errorf("bạn không có quyền xóa tin nhắn này")
		}
	}

	return s.messageRepo.SoftDelete(ctx, messageID)
}

func (s *MessageService) AddReaction(ctx context.Context, userID, messageID string, req dto.AddReactionRequest) (*model.MessageReaction, error) {
	msg, err := s.messageRepo.FindByID(ctx, messageID)
	if err != nil {
		return nil, err
	}

	// Verify user is room member
	_, err = s.roomRepo.FindMember(ctx, msg.RoomId.String(), userID)
	if err != nil {
		return nil, fmt.Errorf("bạn không phải thành viên của phòng này")
	}

	emoji := strings.TrimSpace(req.Emoji)
	if emoji == "" {
		return nil, fmt.Errorf("emoji không được để trống")
	}

	userUUID, _ := uuid.Parse(userID)
	msgUUID, _ := uuid.Parse(messageID)
	now := time.Now()

	reaction := &model.MessageReaction{
		ID:        uuid.New(),
		MessageId: msgUUID,
		UserId:    userUUID,
		Emoji:     emoji,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.messageRepo.AddReaction(ctx, reaction); err != nil {
		return nil, err
	}

	return reaction, nil
}

func (s *MessageService) RemoveReaction(ctx context.Context, userID, messageID, emoji string) error {
	msg, err := s.messageRepo.FindByID(ctx, messageID)
	if err != nil {
		return err
	}

	// Verify user is room member
	_, err = s.roomRepo.FindMember(ctx, msg.RoomId.String(), userID)
	if err != nil {
		return fmt.Errorf("bạn không phải thành viên của phòng này")
	}

	emoji = strings.TrimSpace(emoji)
	if emoji == "" {
		return fmt.Errorf("emoji không được để trống")
	}

	return s.messageRepo.RemoveReaction(ctx, messageID, userID, emoji)
}

func (s *MessageService) MarkAsRead(ctx context.Context, userID, roomID, messageID string) error {
	// Verify user is room member
	_, err := s.roomRepo.FindMember(ctx, roomID, userID)
	if err != nil {
		return fmt.Errorf("bạn không phải thành viên của phòng này")
	}

	return s.messageRepo.MarkRoomMessagesAsRead(ctx, roomID, userID, messageID)
}
