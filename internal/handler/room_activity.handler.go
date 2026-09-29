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

type RoomActivityHandler struct {
	service *service.RoomActivityService
}

func NewRoomActivityHandler(s *service.RoomActivityService) *RoomActivityHandler {
	return &RoomActivityHandler{service: s}
}

func (h *RoomActivityHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware ...gin.HandlerFunc) {
	if len(authMiddleware) > 0 {
		router.Use(authMiddleware...)
	}

	// Tạo và xem danh sách hoạt động theo room
	router.POST("/rooms/:id/activities", h.CreateActivity)
	router.GET("/rooms/:id/activities", h.ListActivities)

	// Xem chi tiết hoạt động
	router.GET("/activities/:id", h.GetActivity)

	// Cập nhật tiến độ / điểm danh / xác nhận của participant
	router.PUT("/activities/:id/participants/:user_id", h.UpdateParticipant)
}

func (h *RoomActivityHandler) CreateActivity(c *gin.Context) {
	userID := middleware.GetUserID(c)
	roomID := c.Param("id")

	var req dto.CreateActivityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrValidation, err.Error())
		return
	}

	res, err := h.service.CreateActivity(c.Request.Context(), userID, roomID, req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusCreated, res, i18n.MsgSuccess)
}

func (h *RoomActivityHandler) ListActivities(c *gin.Context) {
	userID := middleware.GetUserID(c)
	roomID := c.Param("id")

	res, err := h.service.ListActivities(c.Request.Context(), userID, roomID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusOK, res, i18n.MsgSuccess)
}

func (h *RoomActivityHandler) GetActivity(c *gin.Context) {
	userID := middleware.GetUserID(c)
	activityID := c.Param("id")

	res, err := h.service.GetActivity(c.Request.Context(), userID, activityID)
	if err != nil {
		response.Error(c, http.StatusNotFound, i18n.ErrBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusOK, res, i18n.MsgSuccess)
}

func (h *RoomActivityHandler) UpdateParticipant(c *gin.Context) {
	currentUserID := middleware.GetUserID(c)
	activityID := c.Param("id")
	targetUserID := c.Param("user_id")

	var req dto.UpdateParticipantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrValidation, err.Error())
		return
	}

	res, err := h.service.UpdateParticipant(c.Request.Context(), currentUserID, activityID, targetUserID, req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusOK, res, i18n.MsgSuccess)
}
