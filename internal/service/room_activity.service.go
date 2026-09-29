package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"chatapp/internal/dto"
	"chatapp/internal/model"
	"chatapp/internal/repository"
	"chatapp/pkg/eventbus"
)

type RoomActivityService struct {
	activityRepo repository.RoomActivityRepository
	roomRepo     repository.RoomRepository
	messageRepo  repository.MessageRepository
	bus          *eventbus.Bus
}

func NewRoomActivityService(
	activityRepo repository.RoomActivityRepository,
	roomRepo repository.RoomRepository,
	messageRepo repository.MessageRepository,
	bus *eventbus.Bus,
) *RoomActivityService {
	return &RoomActivityService{
		activityRepo: activityRepo,
		roomRepo:     roomRepo,
		messageRepo:  messageRepo,
		bus:          bus,
	}
}

// ──────────────────────────────── Create Activity ────────────────────────────────

func (s *RoomActivityService) CreateActivity(ctx context.Context, creatorIDStr, roomIDStr string, req dto.CreateActivityRequest) (*dto.ActivityResponse, error) {
	creatorUUID, err := uuid.Parse(creatorIDStr)
	if err != nil {
		return nil, fmt.Errorf("creator id không hợp lệ")
	}
	roomUUID, err := uuid.Parse(roomIDStr)
	if err != nil {
		return nil, fmt.Errorf("room id không hợp lệ")
	}

	// Verify membership
	member, err := s.roomRepo.FindMember(ctx, roomIDStr, creatorIDStr)
	if err != nil {
		return nil, fmt.Errorf("bạn không phải thành viên của phòng này")
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, fmt.Errorf("tiêu đề không được để trống")
	}

	actType := strings.ToLower(strings.TrimSpace(req.Type))
	if actType == "" {
		actType = model.ActivityTypeTask
	}

	visibility := strings.ToLower(strings.TrimSpace(req.Visibility))
	if visibility == "" {
		visibility = model.VisibilityPublic
	}

	var dueDate *time.Time
	if req.DueDate != nil && strings.TrimSpace(*req.DueDate) != "" {
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(*req.DueDate))
		if err != nil {
			return nil, fmt.Errorf("định dạng due_date không hợp lệ (RFC3339)")
		}
		dueDate = &t
	}

	activityID := uuid.New()
	now := time.Now()

	// Chuẩn bị danh sách participants và target user IDs cho message
	var targetUUIDs []uuid.UUID
	participants := make([]model.ActivityParticipant, 0, len(req.Participants))

	for _, pInput := range req.Participants {
		uidStr := strings.TrimSpace(pInput.UserID)
		uid, err := uuid.Parse(uidStr)
		if err != nil {
			continue
		}

		status := strings.TrimSpace(pInput.Status)
		if status == "" {
			status = model.ParticipantStatusPending
		}

		participants = append(participants, model.ActivityParticipant{
			ID:           uuid.New(),
			ActivityId:   activityID,
			UserId:       uid,
			Status:       status,
			Note:         strings.TrimSpace(pInput.Note),
			ProgressData: strings.TrimSpace(pInput.ProgressData),
			CreatedAt:    now,
			UpdatedAt:    now,
		})

		targetUUIDs = append(targetUUIDs, uid)
	}

	// Tạo Message card hiển thị trong phòng chat
	cardPayload, _ := json.Marshal(map[string]interface{}{
		"activity_id": activityID.String(),
		"type":        actType,
		"title":       title,
		"visibility":  visibility,
	})

	var msgTargetIDs []uuid.UUID
	isTargetedMsg := false
	if visibility == model.VisibilityTargeted && len(targetUUIDs) > 0 {
		isTargetedMsg = true
		msgTargetIDs = targetUUIDs
	}

	roomMembers, _ := s.roomRepo.FindMembers(ctx, roomIDStr)
	var recipientIDs []uuid.UUID
	for _, m := range roomMembers {
		if m.UserId != creatorUUID {
			if isTargetedMsg {
				for _, tid := range msgTargetIDs {
					if tid == m.UserId {
						recipientIDs = append(recipientIDs, m.UserId)
						break
					}
				}
			} else {
				recipientIDs = append(recipientIDs, m.UserId)
			}
		}
	}

	msgID := uuid.New()
	msg := &model.Message{
		ID:         msgID,
		RoomId:     roomUUID,
		SenderId:   creatorUUID,
		Type:       "activity",
		Content:    string(cardPayload),
		IsTargeted: isTargetedMsg,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	_ = s.messageRepo.Create(ctx, msg, recipientIDs, msgTargetIDs)

	activity := &model.RoomActivity{
		ID:           activityID,
		RoomId:       roomUUID,
		CreatorId:    creatorUUID,
		MessageId:    &msgID,
		Type:         actType,
		Title:        title,
		Description:  strings.TrimSpace(req.Description),
		Visibility:   visibility,
		Metadata:     strings.TrimSpace(req.Metadata),
		DueDate:      dueDate,
		CreatedAt:    now,
		UpdatedAt:    now,
		Participants: participants,
	}

	if err := s.activityRepo.CreateActivity(ctx, activity, participants); err != nil {
		return nil, err
	}

	if s.bus != nil {
		s.bus.Publish(eventbus.Event{
			Type: eventbus.EventActivityCreated,
			Payload: map[string]interface{}{
				"room_id":     roomIDStr,
				"activity_id": activityID.String(),
				"type":        actType,
			},
		})
	}

	isAdmin := member.Role == "owner" || member.Role == "admin"
	return s.toActivityResponse(activity, creatorIDStr, isAdmin), nil
}

// ──────────────────────────────── List Activities ────────────────────────────────

func (s *RoomActivityService) ListActivities(ctx context.Context, userID, roomID string) ([]dto.ActivityResponse, error) {
	member, err := s.roomRepo.FindMember(ctx, roomID, userID)
	if err != nil {
		return nil, fmt.Errorf("bạn không phải thành viên của phòng này")
	}

	isAdmin := member.Role == "owner" || member.Role == "admin"
	activities, err := s.activityRepo.FindActivitiesByRoomID(ctx, roomID, userID, isAdmin)
	if err != nil {
		return nil, err
	}

	result := make([]dto.ActivityResponse, len(activities))
	for i, act := range activities {
		result[i] = *s.toActivityResponse(&act, userID, isAdmin)
	}

	return result, nil
}

// ──────────────────────────────── Get Activity ────────────────────────────────

func (s *RoomActivityService) GetActivity(ctx context.Context, userID, activityID string) (*dto.ActivityResponse, error) {
	act, err := s.activityRepo.FindActivityByID(ctx, activityID)
	if err != nil {
		return nil, fmt.Errorf("hoạt động không tồn tại")
	}

	member, err := s.roomRepo.FindMember(ctx, act.RoomId.String(), userID)
	if err != nil {
		return nil, fmt.Errorf("bạn không phải thành viên của phòng này")
	}

	isAdmin := member.Role == "owner" || member.Role == "admin"
	if act.Visibility == model.VisibilityTargeted && !isAdmin && act.CreatorId.String() != userID {
		// Kiểm tra xem user có phải participant không
		isParticipant := false
		for _, p := range act.Participants {
			if p.UserId.String() == userID {
				isParticipant = true
				break
			}
		}
		if !isParticipant {
			return nil, fmt.Errorf("bạn không có quyền truy cập hoạt động này")
		}
	}

	return s.toActivityResponse(act, userID, isAdmin), nil
}

// ──────────────────────────────── Update Participant Progress ────────────────────────────────

func (s *RoomActivityService) UpdateParticipant(ctx context.Context, currentUserID, activityID, targetUserID string, req dto.UpdateParticipantRequest) (*dto.ActivityParticipantResponse, error) {
	act, err := s.activityRepo.FindActivityByID(ctx, activityID)
	if err != nil {
		return nil, fmt.Errorf("hoạt động không tồn tại")
	}

	member, err := s.roomRepo.FindMember(ctx, act.RoomId.String(), currentUserID)
	if err != nil {
		return nil, fmt.Errorf("bạn không phải thành viên của phòng này")
	}

	isManager := member.Role == "owner" || member.Role == "admin" || act.CreatorId.String() == currentUserID
	isSelf := currentUserID == targetUserID

	if !isManager && !isSelf {
		return nil, fmt.Errorf("bạn chỉ có thể cập nhật trạng thái của chính mình")
	}

	p, err := s.activityRepo.FindParticipant(ctx, activityID, targetUserID)
	if err != nil {
		return nil, fmt.Errorf("người tham gia không tồn tại trong hoạt động này")
	}

	now := time.Now()
	if req.Status != nil {
		p.Status = strings.TrimSpace(*req.Status)
		if p.Status == model.ParticipantStatusSubmitted || p.Status == model.ParticipantStatusConfirmed || p.Status == model.ParticipantStatusPresent {
			p.CompletedAt = &now
		}
	}
	if req.Note != nil {
		p.Note = strings.TrimSpace(*req.Note)
	}
	if req.ProgressData != nil {
		p.ProgressData = strings.TrimSpace(*req.ProgressData)
	}
	if isManager && req.IsVerified != nil && *req.IsVerified {
		verifierUUID, _ := uuid.Parse(currentUserID)
		p.VerifiedBy = &verifierUUID
		p.Status = model.ParticipantStatusConfirmed
	}
	p.UpdatedAt = now

	if err := s.activityRepo.UpdateParticipant(ctx, p); err != nil {
		return nil, err
	}

	if s.bus != nil {
		s.bus.Publish(eventbus.Event{
			Type: eventbus.EventActivityParticipantUpdated,
			Payload: map[string]interface{}{
				"activity_id": activityID,
				"user_id":     targetUserID,
				"status":      p.Status,
			},
		})
	}

	return s.toParticipantResponse(p), nil
}

// ──────────────────────────────── Helpers ────────────────────────────────

func (s *RoomActivityService) toActivityResponse(act *model.RoomActivity, currentUserID string, isAdmin bool) *dto.ActivityResponse {
	var dueDateStr *string
	if act.DueDate != nil {
		t := act.DueDate.Format(time.RFC3339)
		dueDateStr = &t
	}

	var msgIDStr *string
	if act.MessageId != nil {
		m := act.MessageId.String()
		msgIDStr = &m
	}

	// Lọc participants nếu là targeted: người dùng không phải manager thì chỉ thấy phần của mình
	isManager := isAdmin || act.CreatorId.String() == currentUserID
	partResponses := make([]dto.ActivityParticipantResponse, 0, len(act.Participants))

	for _, p := range act.Participants {
		if act.Visibility == model.VisibilityTargeted && !isManager && p.UserId.String() != currentUserID {
			continue // Ẩn thông tin riêng của người khác
		}
		partResponses = append(partResponses, *s.toParticipantResponse(&p))
	}

	return &dto.ActivityResponse{
		ID:           act.ID.String(),
		RoomID:       act.RoomId.String(),
		CreatorID:    act.CreatorId.String(),
		MessageID:    msgIDStr,
		Type:         act.Type,
		Title:        act.Title,
		Description:  act.Description,
		Visibility:   act.Visibility,
		Metadata:     act.Metadata,
		DueDate:      dueDateStr,
		CreatedAt:    act.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    act.UpdatedAt.Format(time.RFC3339),
		Participants: partResponses,
	}
}

func (s *RoomActivityService) toParticipantResponse(p *model.ActivityParticipant) *dto.ActivityParticipantResponse {
	var completedStr *string
	if p.CompletedAt != nil {
		t := p.CompletedAt.Format(time.RFC3339)
		completedStr = &t
	}

	var verifiedStr *string
	if p.VerifiedBy != nil {
		v := p.VerifiedBy.String()
		verifiedStr = &v
	}

	return &dto.ActivityParticipantResponse{
		ID:           p.ID.String(),
		ActivityID:   p.ActivityId.String(),
		UserID:       p.UserId.String(),
		Status:       p.Status,
		Note:         p.Note,
		ProgressData: p.ProgressData,
		CompletedAt:  completedStr,
		VerifiedBy:   verifiedStr,
		CreatedAt:    p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    p.UpdatedAt.Format(time.RFC3339),
	}
}
