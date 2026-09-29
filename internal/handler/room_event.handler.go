package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"chatapp/internal/dto"
	"chatapp/internal/middleware"
	"chatapp/internal/service"
	"chatapp/pkg/i18n"
	"chatapp/pkg/response"
)

type RoomEventHandler struct {
	service *service.RoomEventService
}

func NewRoomEventHandler(s *service.RoomEventService) *RoomEventHandler {
	return &RoomEventHandler{service: s}
}

func (h *RoomEventHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware ...gin.HandlerFunc) {
	if len(authMiddleware) > 0 {
		router.Use(authMiddleware...)
	}

	// ── Pinned Messages ──
	router.POST("/rooms/:id/pins", h.PinMessage)
	router.DELETE("/rooms/:id/pins/:message_id", h.UnpinMessage)
	router.GET("/rooms/:id/pins", h.ListPinnedMessages)

	// ── Polls ──
	router.POST("/rooms/:id/polls", h.CreatePoll)
	router.GET("/polls/:id", h.GetPoll)
	router.POST("/polls/:id/vote", h.VotePoll)

	// ── Room Schedules / Reminders & Calendar View ──
	router.POST("/rooms/:id/schedules", h.CreateSchedule)
	router.GET("/rooms/:id/schedules", h.ListSchedules)
	router.DELETE("/rooms/:id/schedules/:schedule_id", h.DeleteSchedule)
	router.GET("/rooms/:id/calendar", h.GetRoomCalendar) // Bảng lịch theo tháng
}

// ──────────────────────────────── Ghim tin nhắn ────────────────────────────────

func (h *RoomEventHandler) PinMessage(c *gin.Context) {
	userID := middleware.GetUserID(c)
	roomID := c.Param("id")

	var req dto.PinMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrValidation, err.Error())
		return
	}

	res, err := h.service.PinMessage(c.Request.Context(), userID, roomID, req.MessageID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusCreated, res, i18n.MsgSuccess)
}

func (h *RoomEventHandler) UnpinMessage(c *gin.Context) {
	userID := middleware.GetUserID(c)
	roomID := c.Param("id")
	messageID := c.Param("message_id")

	if err := h.service.UnpinMessage(c.Request.Context(), userID, roomID, messageID); err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"status": "unpinned"}, i18n.MsgSuccess)
}

func (h *RoomEventHandler) ListPinnedMessages(c *gin.Context) {
	userID := middleware.GetUserID(c)
	roomID := c.Param("id")

	pins, err := h.service.ListPinnedMessages(c.Request.Context(), userID, roomID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusOK, pins, i18n.MsgSuccess)
}

// ──────────────────────────────── Cuộc bình chọn (Poll) ────────────────────────────────

func (h *RoomEventHandler) CreatePoll(c *gin.Context) {
	userID := middleware.GetUserID(c)
	roomID := c.Param("id")

	var req dto.CreatePollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrValidation, err.Error())
		return
	}

	poll, err := h.service.CreatePoll(c.Request.Context(), userID, roomID, req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusCreated, poll, i18n.MsgSuccess)
}

func (h *RoomEventHandler) GetPoll(c *gin.Context) {
	userID := middleware.GetUserID(c)
	pollID := c.Param("id")

	poll, err := h.service.GetPoll(c.Request.Context(), userID, pollID)
	if err != nil {
		response.Error(c, http.StatusNotFound, i18n.ErrBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusOK, poll, i18n.MsgSuccess)
}

func (h *RoomEventHandler) VotePoll(c *gin.Context) {
	userID := middleware.GetUserID(c)
	pollID := c.Param("id")

	var req dto.VotePollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrValidation, err.Error())
		return
	}

	poll, err := h.service.VotePoll(c.Request.Context(), userID, pollID, req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusOK, poll, i18n.MsgSuccess)
}

// ──────────────────────────────── Đặt lịch thông báo (Schedule) ────────────────────────────────

func (h *RoomEventHandler) CreateSchedule(c *gin.Context) {
	userID := middleware.GetUserID(c)
	roomID := c.Param("id")

	var req dto.CreateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrValidation, err.Error())
		return
	}

	res, err := h.service.CreateSchedule(c.Request.Context(), userID, roomID, req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusCreated, res, i18n.MsgSuccess)
}

func (h *RoomEventHandler) ListSchedules(c *gin.Context) {
	userID := middleware.GetUserID(c)
	roomID := c.Param("id")

	list, err := h.service.ListSchedules(c.Request.Context(), userID, roomID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusOK, list, i18n.MsgSuccess)
}

func (h *RoomEventHandler) DeleteSchedule(c *gin.Context) {
	userID := middleware.GetUserID(c)
	roomID := c.Param("id")
	scheduleID := c.Param("schedule_id")

	if err := h.service.DeleteSchedule(c.Request.Context(), userID, roomID, scheduleID); err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"status": "deleted"}, i18n.MsgSuccess)
}

func (h *RoomEventHandler) GetRoomCalendar(c *gin.Context) {
	userID := middleware.GetUserID(c)
	roomID := c.Param("id")

	var query struct {
		Year  int `form:"year"`
		Month int `form:"month"`
	}
	_ = c.ShouldBindQuery(&query)

	cal, err := h.service.GetRoomCalendar(c.Request.Context(), userID, roomID, query.Year, query.Month)
	if err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusOK, cal, i18n.MsgSuccess)
}
