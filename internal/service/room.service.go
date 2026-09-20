package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"chatapp/internal/dto"
	"chatapp/internal/model"
	"chatapp/internal/repository"
)

type RoomService struct {
	roomRepo repository.RoomRepository
	userRepo repository.UserRepository
}

func NewRoomService(roomRepo repository.RoomRepository, userRepo repository.UserRepository) *RoomService {
	return &RoomService{
		roomRepo: roomRepo,
		userRepo: userRepo,
	}
}

func (s *RoomService) CreateRoom(ctx context.Context, creatorIDStr string, req dto.CreateRoomRequest) (*model.Room, error) {
	creatorID, err := uuid.Parse(creatorIDStr)
	if err != nil {
		return nil, fmt.Errorf("creator id không hợp lệ")
	}

	roomType := strings.ToLower(strings.TrimSpace(req.Type))
	if roomType != "direct" && roomType != "group" {
		return nil, fmt.Errorf("loại phòng chat không hợp lệ")
	}

	now := time.Now()

	if roomType == "direct" {
		if len(req.MemberIDs) != 1 {
			return nil, fmt.Errorf("chat trực tiếp cần đúng 1 người nhận")
		}
		targetIDStr := strings.TrimSpace(req.MemberIDs[0])
		targetID, err := uuid.Parse(targetIDStr)
		if err != nil {
			return nil, fmt.Errorf("id người nhận không hợp lệ")
		}
		if creatorID == targetID {
			return nil, fmt.Errorf("không thể tự chat trực tiếp với chính mình")
		}

		// Check if target user exists
		if _, err := s.userRepo.FindByID(ctx, targetIDStr); err != nil {
			return nil, fmt.Errorf("không tìm thấy người dùng nhận tin: %w", err)
		}

		// Check if direct room already exists
		existing, err := s.roomRepo.FindDirectRoom(ctx, creatorIDStr, targetIDStr)
		if err == nil && existing != nil {
			return existing, nil
		}

		room := &model.Room{
			ID:        uuid.New(),
			Type:      "direct",
			CreatedBy: creatorID,
			CreatedAt: now,
			UpdatedAt: now,
			Members: []model.RoomMember{
				{
					ID:        uuid.New(),
					UserId:    creatorID,
					Role:      "member",
					JoinedAt:  now,
					CreatedAt: now,
					UpdatedAt: now,
				},
				{
					ID:        uuid.New(),
					UserId:    targetID,
					Role:      "member",
					JoinedAt:  now,
					CreatedAt: now,
					UpdatedAt: now,
				},
			},
		}

		if err := s.roomRepo.Create(ctx, room); err != nil {
			return nil, err
		}
		return room, nil
	}

	// Group room
	if req.Name == nil || strings.TrimSpace(*req.Name) == "" {
		return nil, fmt.Errorf("tên nhóm không được để trống")
	}
	name := strings.TrimSpace(*req.Name)

	inviteCode := generateInviteCode()

	room := &model.Room{
		ID:          uuid.New(),
		Type:        "group",
		Name:        &name,
		Description: req.Description,
		AvatarUrl:   req.AvatarUrl,
		CreatedBy:   creatorID,
		InviteCode:  &inviteCode,
		CreatedAt:   now,
		UpdatedAt:   now,
		Members: []model.RoomMember{
			{
				ID:        uuid.New(),
				UserId:    creatorID,
				Role:      "owner",
				JoinedAt:  now,
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}

	// Add other initial members if provided
	for _, memberIDStr := range req.MemberIDs {
		memberIDStr = strings.TrimSpace(memberIDStr)
		if memberIDStr == "" || memberIDStr == creatorIDStr {
			continue
		}
		memberID, err := uuid.Parse(memberIDStr)
		if err != nil {
			continue
		}
		room.Members = append(room.Members, model.RoomMember{
			ID:        uuid.New(),
			UserId:    memberID,
			Role:      "member",
			JoinedAt:  now,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	if err := s.roomRepo.Create(ctx, room); err != nil {
		return nil, err
	}

	return room, nil
}

func (s *RoomService) GetRoom(ctx context.Context, userID, roomID string) (*model.Room, error) {
	// Check user is member
	_, err := s.roomRepo.FindMember(ctx, roomID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("bạn không phải thành viên của phòng này")
		}
		return nil, err
	}

	return s.roomRepo.FindByID(ctx, roomID)
}

func (s *RoomService) ListUserRooms(ctx context.Context, userID string, page, perPage int) ([]model.Room, int64, error) {
	if page < 1 {
		page = 1
	}
	if perPage <= 0 || perPage > 50 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	return s.roomRepo.FindByUserID(ctx, userID, perPage, offset)
}

func (s *RoomService) UpdateRoom(ctx context.Context, userID, roomID string, req dto.UpdateRoomRequest) (*model.Room, error) {
	member, err := s.roomRepo.FindMember(ctx, roomID, userID)
	if err != nil {
		return nil, fmt.Errorf("bạn không phải thành viên của phòng này")
	}

	room, err := s.roomRepo.FindByID(ctx, roomID)
	if err != nil {
		return nil, err
	}

	if room.Type == "group" && member.Role != "owner" && member.Role != "admin" {
		return nil, fmt.Errorf("chỉ quản trị viên mới có quyền đổi thông tin nhóm")
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, fmt.Errorf("tên nhóm không được để trống")
		}
		room.Name = &name
	}

	if req.Description != nil {
		room.Description = req.Description
	}

	if req.AvatarUrl != nil {
		room.AvatarUrl = req.AvatarUrl
	}

	room.UpdatedAt = time.Now()
	if err := s.roomRepo.Update(ctx, room); err != nil {
		return nil, err
	}

	return room, nil
}

func (s *RoomService) DeleteRoom(ctx context.Context, userID, roomID string) error {
	member, err := s.roomRepo.FindMember(ctx, roomID, userID)
	if err != nil {
		return fmt.Errorf("bạn không phải thành viên của phòng này")
	}

	room, err := s.roomRepo.FindByID(ctx, roomID)
	if err != nil {
		return err
	}

	if room.Type == "group" && member.Role != "owner" && room.CreatedBy.String() != userID {
		return fmt.Errorf("chỉ trưởng nhóm mới có quyền giải tán nhóm")
	}

	return s.roomRepo.Delete(ctx, roomID)
}

func (s *RoomService) AddMember(ctx context.Context, currentUserID, roomID string, req dto.AddRoomMemberRequest) (*model.RoomMember, error) {
	// Check current user permission
	currentMember, err := s.roomRepo.FindMember(ctx, roomID, currentUserID)
	if err != nil {
		return nil, fmt.Errorf("bạn không phải thành viên của phòng này")
	}

	room, err := s.roomRepo.FindByID(ctx, roomID)
	if err != nil {
		return nil, err
	}

	if room.Type == "direct" {
		return nil, fmt.Errorf("không thể thêm thành viên vào cuộc trò chuyện 1-1")
	}

	if currentMember.Role != "owner" && currentMember.Role != "admin" {
		return nil, fmt.Errorf("chỉ quản trị viên mới có thể thêm thành viên")
	}

	targetUserIDStr := strings.TrimSpace(req.UserID)
	targetUUID, err := uuid.Parse(targetUserIDStr)
	if err != nil {
		return nil, fmt.Errorf("id người dùng không hợp lệ")
	}

	// Verify target user exists
	if _, err := s.userRepo.FindByID(ctx, targetUserIDStr); err != nil {
		return nil, fmt.Errorf("không tìm thấy người dùng: %w", err)
	}

	// Check if already in room
	_, err = s.roomRepo.FindMember(ctx, roomID, targetUserIDStr)
	if err == nil {
		return nil, fmt.Errorf("người dùng đã là thành viên của nhóm")
	}

	role := "member"
	if req.Role != nil && (*req.Role == "admin" || *req.Role == "member") {
		role = *req.Role
	}

	roomUUID, _ := uuid.Parse(roomID)
	now := time.Now()
	newMember := &model.RoomMember{
		ID:        uuid.New(),
		RoomId:    roomUUID,
		UserId:    targetUUID,
		Role:      role,
		Nickname:  req.Nickname,
		JoinedAt:  now,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.roomRepo.AddMember(ctx, newMember); err != nil {
		return nil, err
	}

	return newMember, nil
}

func (s *RoomService) RemoveMember(ctx context.Context, currentUserID, roomID, targetUserID string) error {
	currentMember, err := s.roomRepo.FindMember(ctx, roomID, currentUserID)
	if err != nil {
		return fmt.Errorf("bạn không phải thành viên của phòng này")
	}

	if currentMember.Role != "owner" && currentMember.Role != "admin" {
		return fmt.Errorf("chỉ quản trị viên mới có quyền xóa thành viên")
	}

	targetMember, err := s.roomRepo.FindMember(ctx, roomID, targetUserID)
	if err != nil {
		return fmt.Errorf("không tìm thấy thành viên trong nhóm")
	}

	if targetMember.Role == "owner" {
		return fmt.Errorf("không thể xóa trưởng nhóm")
	}

	if currentMember.Role == "admin" && targetMember.Role == "admin" {
		return fmt.Errorf("quản trị viên không thể xóa quản trị viên khác")
	}

	return s.roomRepo.RemoveMember(ctx, roomID, targetUserID)
}

func (s *RoomService) LeaveRoom(ctx context.Context, userID, roomID string) error {
	member, err := s.roomRepo.FindMember(ctx, roomID, userID)
	if err != nil {
		return fmt.Errorf("bạn không phải thành viên của phòng này")
	}

	room, err := s.roomRepo.FindByID(ctx, roomID)
	if err != nil {
		return err
	}

	if room.Type == "direct" {
		return fmt.Errorf("không thể rời khỏi cuộc trò chuyện trực tiếp")
	}

	if member.Role == "owner" {
		count, _ := s.roomRepo.CountMembers(ctx, roomID)
		if count > 1 {
			return fmt.Errorf("vui lòng chuyển quyền trưởng nhóm trước khi rời nhóm")
		}
		// Last member leaves -> delete room
		return s.roomRepo.Delete(ctx, roomID)
	}

	return s.roomRepo.RemoveMember(ctx, roomID, userID)
}

func (s *RoomService) UpdateMember(ctx context.Context, currentUserID, roomID, targetUserID string, req dto.UpdateRoomMemberRequest) (*model.RoomMember, error) {
	currentMember, err := s.roomRepo.FindMember(ctx, roomID, currentUserID)
	if err != nil {
		return nil, fmt.Errorf("bạn không phải thành viên của phòng này")
	}

	targetMember, err := s.roomRepo.FindMember(ctx, roomID, targetUserID)
	if err != nil {
		return nil, fmt.Errorf("không tìm thấy thành viên trong nhóm")
	}

	if req.Role != nil {
		if currentMember.Role != "owner" {
			return nil, fmt.Errorf("chỉ trưởng nhóm mới có quyền thay đổi vai trò")
		}
		newRole := *req.Role
		if newRole != "owner" && newRole != "admin" && newRole != "member" {
			return nil, fmt.Errorf("vai trò không hợp lệ")
		}
		if newRole == "owner" {
			// Transfer ownership: current user becomes admin
			currentMember.Role = "admin"
			currentMember.UpdatedAt = time.Now()
			_ = s.roomRepo.UpdateMember(ctx, currentMember)
		}
		targetMember.Role = newRole
	}

	if req.Nickname != nil {
		targetMember.Nickname = req.Nickname
	}

	targetMember.UpdatedAt = time.Now()
	if err := s.roomRepo.UpdateMember(ctx, targetMember); err != nil {
		return nil, err
	}

	return targetMember, nil
}

func (s *RoomService) UpdateMyMemberSettings(ctx context.Context, userID, roomID string, req dto.UpdateRoomMemberRequest) (*model.RoomMember, error) {
	member, err := s.roomRepo.FindMember(ctx, roomID, userID)
	if err != nil {
		return nil, fmt.Errorf("bạn không phải thành viên của phòng này")
	}

	if req.Nickname != nil {
		member.Nickname = req.Nickname
	}
	if req.IsMuted != nil {
		member.IsMuted = *req.IsMuted
	}
	if req.IsArchived != nil {
		member.IsArchived = *req.IsArchived
	}
	if req.IsPinned != nil {
		member.IsPinned = *req.IsPinned
	}

	member.UpdatedAt = time.Now()
	if err := s.roomRepo.UpdateMember(ctx, member); err != nil {
		return nil, err
	}

	return member, nil
}

func (s *RoomService) JoinByInviteCode(ctx context.Context, userID, code string) (*model.Room, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("id người dùng không hợp lệ")
	}

	code = strings.TrimSpace(code)
	if code == "" {
		return nil, fmt.Errorf("mã mời không được để trống")
	}

	room, err := s.roomRepo.FindByInviteCode(ctx, code)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("mã mời không hợp lệ hoặc nhóm không tồn tại")
		}
		return nil, err
	}

	// Check if already member
	_, err = s.roomRepo.FindMember(ctx, room.ID.String(), userID)
	if err == nil {
		// Already member, return room
		return room, nil
	}

	now := time.Now()
	member := &model.RoomMember{
		ID:        uuid.New(),
		RoomId:    room.ID,
		UserId:    userUUID,
		Role:      "member",
		JoinedAt:  now,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.roomRepo.AddMember(ctx, member); err != nil {
		return nil, err
	}

	room.Members = append(room.Members, *member)
	return room, nil
}

func generateInviteCode() string {
	b := make([]byte, 5)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
