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

type RoomEventService struct {
	eventRepo   repository.RoomEventRepository
	roomRepo    repository.RoomRepository
	messageRepo repository.MessageRepository
	bus         *eventbus.Bus
}

func NewRoomEventService(
	eventRepo repository.RoomEventRepository,
	roomRepo repository.RoomRepository,
	messageRepo repository.MessageRepository,
	bus *eventbus.Bus,
) *RoomEventService {
	return &RoomEventService{
		eventRepo:   eventRepo,
		roomRepo:    roomRepo,
		messageRepo: messageRepo,
		bus:         bus,
	}
}

// ──────────────────────────────── Ghim tin nhắn (Pin) ────────────────────────────────

func (s *RoomEventService) PinMessage(ctx context.Context, userID, roomID, messageID string) (*dto.PinnedMessageResponse, error) {
	// Verify membership
	_, err := s.roomRepo.FindMember(ctx, roomID, userID)
	if err != nil {
		return nil, fmt.Errorf("bạn không phải thành viên của phòng này")
	}

	msg, err := s.messageRepo.FindByID(ctx, messageID)
	if err != nil {
		return nil, fmt.Errorf("tin nhắn không tồn tại")
	}
	if msg.RoomId.String() != roomID {
		return nil, fmt.Errorf("tin nhắn không thuộc phòng chat này")
	}

	isPinned, err := s.eventRepo.IsMessagePinned(ctx, roomID, messageID)
	if err != nil {
		return nil, err
	}
	if isPinned {
		return nil, fmt.Errorf("tin nhắn này đã được ghim từ trước")
	}

	userUUID, _ := uuid.Parse(userID)
	roomUUID, _ := uuid.Parse(roomID)
	msgUUID, _ := uuid.Parse(messageID)

	pinned := &model.PinnedMessage{
		ID:        uuid.New(),
		RoomId:    roomUUID,
		MessageId: msgUUID,
		PinnedBy:  userUUID,
		PinnedAt:  time.Now(),
	}

	if err := s.eventRepo.PinMessage(ctx, pinned); err != nil {
		return nil, err
	}

	if s.bus != nil {
		s.bus.Publish(eventbus.Event{
			Type: eventbus.EventMessagePinned,
			Payload: map[string]interface{}{
				"room_id":    roomID,
				"message_id": messageID,
				"pinned_by":  userID,
			},
		})
	}

	return &dto.PinnedMessageResponse{
		ID:        pinned.ID.String(),
		RoomID:    roomID,
		MessageID: messageID,
		PinnedBy:  userID,
		PinnedAt:  pinned.PinnedAt.Format(time.RFC3339),
	}, nil
}

func (s *RoomEventService) UnpinMessage(ctx context.Context, userID, roomID, messageID string) error {
	_, err := s.roomRepo.FindMember(ctx, roomID, userID)
	if err != nil {
		return fmt.Errorf("bạn không phải thành viên của phòng này")
	}

	if err := s.eventRepo.UnpinMessage(ctx, roomID, messageID); err != nil {
		return err
	}

	if s.bus != nil {
		s.bus.Publish(eventbus.Event{
			Type: eventbus.EventMessageUnpinned,
			Payload: map[string]interface{}{
				"room_id":     roomID,
				"message_id":  messageID,
				"unpinned_by": userID,
			},
		})
	}

	return nil
}

func (s *RoomEventService) ListPinnedMessages(ctx context.Context, userID, roomID string) ([]dto.PinnedMessageResponse, error) {
	_, err := s.roomRepo.FindMember(ctx, roomID, userID)
	if err != nil {
		return nil, fmt.Errorf("bạn không phải thành viên của phòng này")
	}

	pins, err := s.eventRepo.FindPinnedMessages(ctx, roomID)
	if err != nil {
		return nil, err
	}

	result := make([]dto.PinnedMessageResponse, len(pins))
	for i, p := range pins {
		var msgRes *dto.MessageResponse
		if p.Message != nil {
			msgRes = &dto.MessageResponse{
				ID:        p.Message.ID.String(),
				RoomID:    p.Message.RoomId.String(),
				SenderID:  p.Message.SenderId.String(),
				Type:      p.Message.Type,
				Content:   p.Message.Content,
				IsEdited:  p.Message.IsEdited,
				IsDeleted: p.Message.IsDeleted,
				CreatedAt: p.Message.CreatedAt.Format(time.RFC3339),
				UpdatedAt: p.Message.UpdatedAt.Format(time.RFC3339),
			}
		}
		result[i] = dto.PinnedMessageResponse{
			ID:        p.ID.String(),
			RoomID:    p.RoomId.String(),
			MessageID: p.MessageId.String(),
			PinnedBy:  p.PinnedBy.String(),
			PinnedAt:  p.PinnedAt.Format(time.RFC3339),
			Message:   msgRes,
		}
	}

	return result, nil
}

// ──────────────────────────────── Bình chọn (Poll) ────────────────────────────────

func (s *RoomEventService) CreatePoll(ctx context.Context, userID, roomID string, req dto.CreatePollRequest) (*dto.PollResponse, error) {
	_, err := s.roomRepo.FindMember(ctx, roomID, userID)
	if err != nil {
		return nil, fmt.Errorf("bạn không phải thành viên của phòng này")
	}

	question := strings.TrimSpace(req.Question)
	if question == "" {
		return nil, fmt.Errorf("câu hỏi bình chọn không được để trống")
	}
	if len(req.Options) < 2 {
		return nil, fmt.Errorf("cuộc bình chọn phải có ít nhất 2 phương án")
	}

	userUUID, _ := uuid.Parse(userID)
	roomUUID, _ := uuid.Parse(roomID)
	now := time.Now()

	var expiresAt *time.Time
	if req.ExpiresAt != nil && strings.TrimSpace(*req.ExpiresAt) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*req.ExpiresAt))
		if err != nil {
			return nil, fmt.Errorf("định dạng expires_at không hợp lệ (RFC3339)")
		}
		if parsed.Before(now) {
			return nil, fmt.Errorf("thời gian kết thúc phải ở tương lai")
		}
		expiresAt = &parsed
	}

	pollID := uuid.New()
	pollOptions := make([]model.PollOption, len(req.Options))
	for i, optText := range req.Options {
		pollOptions[i] = model.PollOption{
			ID:     uuid.New(),
			PollId: pollID,
			Text:   strings.TrimSpace(optText),
			Order:  i,
		}
	}

	// Tạo Message đại diện cho poll trong phòng chat
	pollPayloadBytes, _ := json.Marshal(map[string]interface{}{
		"poll_id":  pollID.String(),
		"question": question,
	})
	pollMsg := &model.Message{
		ID:        uuid.New(),
		RoomId:    roomUUID,
		SenderId:  userUUID,
		Type:      "poll",
		Content:   string(pollPayloadBytes),
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Lấy danh sách thành viên để update status message
	members, _ := s.roomRepo.FindMembers(ctx, roomID)
	var recipientIDs []uuid.UUID
	for _, m := range members {
		if m.UserId != userUUID {
			recipientIDs = append(recipientIDs, m.UserId)
		}
	}
	_ = s.messageRepo.Create(ctx, pollMsg, recipientIDs, nil)

	poll := &model.Poll{
		ID:          pollID,
		RoomId:      roomUUID,
		MessageId:   &pollMsg.ID,
		CreatorId:   userUUID,
		Question:    question,
		IsMultiple:  req.IsMultiple,
		IsAnonymous: req.IsAnonymous,
		ExpiresAt:   expiresAt,
		CreatedAt:   now,
		UpdatedAt:   now,
		Options:     pollOptions,
	}

	if err := s.eventRepo.CreatePoll(ctx, poll); err != nil {
		return nil, err
	}

	if s.bus != nil {
		s.bus.Publish(eventbus.Event{
			Type: eventbus.EventPollCreated,
			Payload: map[string]interface{}{
				"room_id": roomID,
				"poll_id": pollID.String(),
			},
		})
	}

	return s.toPollResponse(poll, userID), nil
}

func (s *RoomEventService) GetPoll(ctx context.Context, userID, pollID string) (*dto.PollResponse, error) {
	poll, err := s.eventRepo.FindPollByID(ctx, pollID)
	if err != nil {
		return nil, fmt.Errorf("cuộc bình chọn không tồn tại")
	}

	_, err = s.roomRepo.FindMember(ctx, poll.RoomId.String(), userID)
	if err != nil {
		return nil, fmt.Errorf("bạn không phải thành viên của phòng này")
	}

	return s.toPollResponse(poll, userID), nil
}

func (s *RoomEventService) VotePoll(ctx context.Context, userID, pollID string, req dto.VotePollRequest) (*dto.PollResponse, error) {
	poll, err := s.eventRepo.FindPollByID(ctx, pollID)
	if err != nil {
		return nil, fmt.Errorf("cuộc bình chọn không tồn tại")
	}

	_, err = s.roomRepo.FindMember(ctx, poll.RoomId.String(), userID)
	if err != nil {
		return nil, fmt.Errorf("bạn không phải thành viên của phòng này")
	}

	if poll.ExpiresAt != nil && time.Now().After(*poll.ExpiresAt) {
		return nil, fmt.Errorf("cuộc bình chọn đã kết thúc")
	}

	if !poll.IsMultiple && len(req.OptionIDs) > 1 {
		return nil, fmt.Errorf("cuộc bình chọn này chỉ cho phép chọn 1 phương án")
	}

	userUUID, _ := uuid.Parse(userID)
	optionUUIDs := make([]uuid.UUID, 0, len(req.OptionIDs))
	for _, optStr := range req.OptionIDs {
		parsed, err := uuid.Parse(optStr)
		if err == nil {
			optionUUIDs = append(optionUUIDs, parsed)
		}
	}

	if err := s.eventRepo.VotePoll(ctx, pollID, userUUID, optionUUIDs, poll.IsMultiple); err != nil {
		return nil, err
	}

	if s.bus != nil {
		s.bus.Publish(eventbus.Event{
			Type: eventbus.EventPollVoted,
			Payload: map[string]interface{}{
				"poll_id": pollID,
				"user_id": userID,
			},
		})
	}

	// Lấy lại poll với lượt vote mới
	updatedPoll, err := s.eventRepo.FindPollByID(ctx, pollID)
	if err != nil {
		return nil, err
	}

	return s.toPollResponse(updatedPoll, userID), nil
}

func (s *RoomEventService) toPollResponse(poll *model.Poll, userID string) *dto.PollResponse {
	userUUID, _ := uuid.Parse(userID)
	totalVotes := 0

	optResponses := make([]dto.PollOptionResponse, len(poll.Options))
	for i, opt := range poll.Options {
		voteCount := len(opt.Votes)
		totalVotes += voteCount

		hasVoted := false
		var voterIDs []string
		for _, v := range opt.Votes {
			if v.UserId == userUUID {
				hasVoted = true
			}
			if !poll.IsAnonymous {
				voterIDs = append(voterIDs, v.UserId.String())
			}
		}

		optResponses[i] = dto.PollOptionResponse{
			ID:        opt.ID.String(),
			Text:      opt.Text,
			VoteCount: voteCount,
			VoterIDs:  voterIDs,
			HasVoted:  hasVoted,
		}
	}

	var expiresStr *string
	if poll.ExpiresAt != nil {
		t := poll.ExpiresAt.Format(time.RFC3339)
		expiresStr = &t
	}

	var msgIDStr *string
	if poll.MessageId != nil {
		s := poll.MessageId.String()
		msgIDStr = &s
	}

	return &dto.PollResponse{
		ID:          poll.ID.String(),
		RoomID:      poll.RoomId.String(),
		MessageID:   msgIDStr,
		CreatorID:   poll.CreatorId.String(),
		Question:    poll.Question,
		IsMultiple:  poll.IsMultiple,
		IsAnonymous: poll.IsAnonymous,
		TotalVotes:  totalVotes,
		Options:     optResponses,
		ExpiresAt:   expiresStr,
		CreatedAt:   poll.CreatedAt.Format(time.RFC3339),
	}
}

// ──────────────────────────────── Đặt lịch thông báo / Nhắc hẹn (Schedule) ────────────────────────────────

func (s *RoomEventService) CreateSchedule(ctx context.Context, userID, roomID string, req dto.CreateScheduleRequest) (*dto.ScheduleResponse, error) {
	_, err := s.roomRepo.FindMember(ctx, roomID, userID)
	if err != nil {
		return nil, fmt.Errorf("bạn không phải thành viên của phòng này")
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, fmt.Errorf("tiêu đề lịch hẹn không được để trống")
	}

	eventTime, err := time.Parse(time.RFC3339, strings.TrimSpace(req.EventTime))
	if err != nil {
		return nil, fmt.Errorf("định dạng event_time không hợp lệ (RFC3339)")
	}

	var remindAt *time.Time
	if req.RemindAt != nil && strings.TrimSpace(*req.RemindAt) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*req.RemindAt))
		if err != nil {
			return nil, fmt.Errorf("định dạng remind_at không hợp lệ (RFC3339)")
		}
		remindAt = &parsed
	}

	userUUID, _ := uuid.Parse(userID)
	roomUUID, _ := uuid.Parse(roomID)
	now := time.Now()

	var endTime *time.Time
	if req.EndTime != nil && strings.TrimSpace(*req.EndTime) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*req.EndTime))
		if err == nil {
			endTime = &parsed
		}
	}

	color := strings.TrimSpace(req.Color)
	if color == "" {
		color = "#3B82F6"
	}

	category := strings.TrimSpace(req.Category)
	if category == "" {
		category = "general"
	}

	schedule := &model.RoomSchedule{
		ID:          uuid.New(),
		RoomId:      roomUUID,
		CreatorId:   userUUID,
		Title:       title,
		Description: strings.TrimSpace(req.Description),
		Category:    category,
		EventTime:   eventTime,
		EndTime:     endTime,
		IsAllDay:    req.IsAllDay,
		Color:       color,
		Location:    strings.TrimSpace(req.Location),
		RemindAt:    remindAt,
		IsNotified:  false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.eventRepo.CreateSchedule(ctx, schedule); err != nil {
		return nil, err
	}

	// Gửi tin nhắn loại schedule vào phòng chat để các thành viên nhận thông báo
	schedPayloadBytes, _ := json.Marshal(map[string]interface{}{
		"schedule_id": schedule.ID.String(),
		"title":       schedule.Title,
		"category":    schedule.Category,
		"event_time":  schedule.EventTime.Format(time.RFC3339),
		"location":    schedule.Location,
	})
	schedMsg := &model.Message{
		ID:        uuid.New(),
		RoomId:    roomUUID,
		SenderId:  userUUID,
		Type:      "schedule",
		Content:   string(schedPayloadBytes),
		CreatedAt: now,
		UpdatedAt: now,
	}
	members, _ := s.roomRepo.FindMembers(ctx, roomID)
	var recipientIDs []uuid.UUID
	for _, m := range members {
		if m.UserId != userUUID {
			recipientIDs = append(recipientIDs, m.UserId)
		}
	}
	_ = s.messageRepo.Create(ctx, schedMsg, recipientIDs, nil)

	if s.bus != nil {
		s.bus.Publish(eventbus.Event{
			Type: eventbus.EventScheduleCreated,
			Payload: map[string]interface{}{
				"room_id":     roomID,
				"schedule_id": schedule.ID.String(),
				"title":       schedule.Title,
			},
		})
	}

	return s.toScheduleResponse(schedule), nil
}

func (s *RoomEventService) ListSchedules(ctx context.Context, userID, roomID string) ([]dto.ScheduleResponse, error) {
	_, err := s.roomRepo.FindMember(ctx, roomID, userID)
	if err != nil {
		return nil, fmt.Errorf("bạn không phải thành viên của phòng này")
	}

	list, err := s.eventRepo.FindSchedulesByRoomID(ctx, roomID)
	if err != nil {
		return nil, err
	}

	result := make([]dto.ScheduleResponse, len(list))
	for i, item := range list {
		result[i] = *s.toScheduleResponse(&item)
	}

	return result, nil
}

// GetRoomCalendar trả về toàn bộ bảng lịch của phòng theo tháng (dù không có lịch vẫn có đủ các ngày trong tháng)
func (s *RoomEventService) GetRoomCalendar(ctx context.Context, userID, roomID string, year, month int) (*dto.RoomCalendarResponse, error) {
	_, err := s.roomRepo.FindMember(ctx, roomID, userID)
	if err != nil {
		return nil, fmt.Errorf("bạn không phải thành viên của phòng này")
	}

	now := time.Now()
	if year <= 0 {
		year = now.Year()
	}
	if month < 1 || month > 12 {
		month = int(now.Month())
	}

	loc := now.Location()
	monthStart := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, loc)
	monthEnd := monthStart.AddDate(0, 1, 0).Add(-time.Nanosecond)
	totalDays := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, loc).Day()

	schedules, err := s.eventRepo.FindSchedulesByDateRange(ctx, roomID, monthStart, monthEnd)
	if err != nil {
		return nil, err
	}

	todayStr := now.Format("2006-01-02")
	todayDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	dayOfWeekNames := map[int]string{
		1: "Thứ Hai",
		2: "Thứ Ba",
		3: "Thứ Tư",
		4: "Thứ Năm",
		5: "Thứ Sáu",
		6: "Thứ Bảy",
		7: "Chủ Nhật",
	}

	days := make([]dto.CalendarDayResponse, totalDays)
	totalEventsInMonth := 0

	for d := 1; d <= totalDays; d++ {
		currentDate := time.Date(year, time.Month(month), d, 0, 0, 0, 0, loc)
		dateStr := fmt.Sprintf("%04d-%02d-%02d", year, month, d)

		weekday := int(currentDate.Weekday())
		if weekday == 0 {
			weekday = 7
		}

		dayEvents := make([]dto.CalendarItemResponse, 0)
		for _, sc := range schedules {
			scDateStr := sc.EventTime.In(loc).Format("2006-01-02")
			if scDateStr == dateStr {
				var endTimeStr *string
				if sc.EndTime != nil {
					t := sc.EndTime.Format(time.RFC3339)
					endTimeStr = &t
				}
				dayEvents = append(dayEvents, dto.CalendarItemResponse{
					ID:        sc.ID.String(),
					Source:    "schedule",
					Type:      sc.Category,
					Title:     sc.Title,
					StartTime: sc.EventTime.Format(time.RFC3339),
					EndTime:   endTimeStr,
					IsAllDay:  sc.IsAllDay,
					Color:     sc.Color,
					Location:  sc.Location,
				})
			}
		}

		totalEventsInMonth += len(dayEvents)

		days[d-1] = dto.CalendarDayResponse{
			Date:          dateStr,
			Day:           d,
			DayOfWeek:     weekday,
			DayOfWeekName: dayOfWeekNames[weekday],
			IsToday:       dateStr == todayStr,
			IsPast:        currentDate.Before(todayDate),
			TotalEvents:   len(dayEvents),
			Events:        dayEvents,
		}
	}

	return &dto.RoomCalendarResponse{
		RoomID:             roomID,
		Year:               year,
		Month:              month,
		Today:              todayStr,
		TotalDays:          totalDays,
		TotalEventsInMonth: totalEventsInMonth,
		Days:               days,
	}, nil
}

func (s *RoomEventService) toScheduleResponse(sc *model.RoomSchedule) *dto.ScheduleResponse {
	var remindStr *string
	if sc.RemindAt != nil {
		t := sc.RemindAt.Format(time.RFC3339)
		remindStr = &t
	}

	var endTimeStr *string
	if sc.EndTime != nil {
		t := sc.EndTime.Format(time.RFC3339)
		endTimeStr = &t
	}

	return &dto.ScheduleResponse{
		ID:          sc.ID.String(),
		RoomID:      sc.RoomId.String(),
		CreatorID:   sc.CreatorId.String(),
		Title:       sc.Title,
		Description: sc.Description,
		Category:    sc.Category,
		EventTime:   sc.EventTime.Format(time.RFC3339),
		EndTime:     endTimeStr,
		IsAllDay:    sc.IsAllDay,
		Color:       sc.Color,
		Location:    sc.Location,
		RemindAt:    remindStr,
		IsNotified:  sc.IsNotified,
		CreatedAt:   sc.CreatedAt.Format(time.RFC3339),
	}
}

func (s *RoomEventService) DeleteSchedule(ctx context.Context, userID, roomID, scheduleID string) error {
	sched, err := s.eventRepo.FindScheduleByID(ctx, scheduleID)
	if err != nil {
		return fmt.Errorf("lịch hẹn không tồn tại")
	}

	if sched.CreatorId.String() != userID {
		member, err := s.roomRepo.FindMember(ctx, roomID, userID)
		if err != nil || (member.Role != "owner" && member.Role != "admin") {
			return fmt.Errorf("bạn không có quyền xóa lịch hẹn này")
		}
	}

	if err := s.eventRepo.DeleteSchedule(ctx, scheduleID, roomID); err != nil {
		return err
	}

	if s.bus != nil {
		s.bus.Publish(eventbus.Event{
			Type: eventbus.EventScheduleDeleted,
			Payload: map[string]interface{}{
				"room_id":     roomID,
				"schedule_id": scheduleID,
			},
		})
	}

	return nil
}
