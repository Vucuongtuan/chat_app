package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"chatapp/internal/dto"
	"chatapp/internal/middleware"
	"chatapp/internal/model"
	"chatapp/internal/service"
	"chatapp/pkg/i18n"
	"chatapp/pkg/response"
)

type RoomHandler struct {
	service *service.RoomService
}

func NewRoomHandler(s *service.RoomService) *RoomHandler {
	return &RoomHandler{service: s}
}

func (h *RoomHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware ...gin.HandlerFunc) {
	rooms := router.Group("/rooms")
	if len(authMiddleware) > 0 {
		rooms.Use(authMiddleware...)
	}
	{
		rooms.POST("", h.CreateRoom)
		rooms.GET("", h.ListUserRooms)
		rooms.POST("/join", h.JoinByInviteCode)
		rooms.GET("/:id", h.GetRoom)
		rooms.PUT("/:id", h.UpdateRoom)
		rooms.DELETE("/:id", h.DeleteRoom)
		rooms.POST("/:id/leave", h.LeaveRoom)
		rooms.PUT("/:id/me", h.UpdateMyMemberSettings)
		rooms.POST("/:id/members", h.AddMember)
		rooms.DELETE("/:id/members/:userId", h.RemoveMember)
		rooms.PUT("/:id/members/:userId", h.UpdateMember)
	}
}

func (h *RoomHandler) CreateRoom(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, i18n.ErrUnauthorized)
		return
	}

	var req dto.CreateRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrValidation, err.Error())
		return
	}

	room, err := h.service.CreateRoom(c.Request.Context(), userID, req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusCreated, toRoomResponse(room), i18n.MsgSuccess)
}

func (h *RoomHandler) GetRoom(c *gin.Context) {
	userID := middleware.GetUserID(c)
	roomID := c.Param("id")

	room, err := h.service.GetRoom(c.Request.Context(), userID, roomID)
	if err != nil {
		response.Error(c, http.StatusForbidden, i18n.ErrForbidden, err.Error())
		return
	}

	response.Success(c, http.StatusOK, toRoomResponse(room), i18n.MsgSuccess)
}

func (h *RoomHandler) ListUserRooms(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, i18n.ErrUnauthorized)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	rooms, total, err := h.service.ListUserRooms(c.Request.Context(), userID, page, perPage)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, i18n.ErrServer, err.Error())
		return
	}

	result := make([]dto.RoomResponse, len(rooms))
	for i := range rooms {
		result[i] = toRoomResponse(&rooms[i])
	}

	response.Success(c, http.StatusOK, gin.H{"items": result, "total": total, "page": page, "per_page": perPage}, i18n.MsgSuccess)
}

func (h *RoomHandler) UpdateRoom(c *gin.Context) {
	userID := middleware.GetUserID(c)
	roomID := c.Param("id")

	var req dto.UpdateRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrValidation, err.Error())
		return
	}

	room, err := h.service.UpdateRoom(c.Request.Context(), userID, roomID, req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusOK, toRoomResponse(room), i18n.MsgSuccess)
}

func (h *RoomHandler) DeleteRoom(c *gin.Context) {
	userID := middleware.GetUserID(c)
	roomID := c.Param("id")

	if err := h.service.DeleteRoom(c.Request.Context(), userID, roomID); err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrBadRequest, err.Error())
		return
	}

	response.Message(c, http.StatusOK, i18n.MsgSuccess)
}

func (h *RoomHandler) AddMember(c *gin.Context) {
	userID := middleware.GetUserID(c)
	roomID := c.Param("id")

	var req dto.AddRoomMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrValidation, err.Error())
		return
	}

	member, err := h.service.AddMember(c.Request.Context(), userID, roomID, req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusCreated, toRoomMemberResponse(member), i18n.MsgSuccess)
}

func (h *RoomHandler) RemoveMember(c *gin.Context) {
	userID := middleware.GetUserID(c)
	roomID := c.Param("id")
	targetUserID := c.Param("userId")

	if err := h.service.RemoveMember(c.Request.Context(), userID, roomID, targetUserID); err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrBadRequest, err.Error())
		return
	}

	response.Message(c, http.StatusOK, i18n.MsgSuccess)
}

func (h *RoomHandler) LeaveRoom(c *gin.Context) {
	userID := middleware.GetUserID(c)
	roomID := c.Param("id")

	if err := h.service.LeaveRoom(c.Request.Context(), userID, roomID); err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrBadRequest, err.Error())
		return
	}

	response.Message(c, http.StatusOK, i18n.MsgSuccess)
}

func (h *RoomHandler) UpdateMember(c *gin.Context) {
	userID := middleware.GetUserID(c)
	roomID := c.Param("id")
	targetUserID := c.Param("userId")

	var req dto.UpdateRoomMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrValidation, err.Error())
		return
	}

	member, err := h.service.UpdateMember(c.Request.Context(), userID, roomID, targetUserID, req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusOK, toRoomMemberResponse(member), i18n.MsgSuccess)
}

func (h *RoomHandler) UpdateMyMemberSettings(c *gin.Context) {
	userID := middleware.GetUserID(c)
	roomID := c.Param("id")

	var req dto.UpdateRoomMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrValidation, err.Error())
		return
	}

	member, err := h.service.UpdateMyMemberSettings(c.Request.Context(), userID, roomID, req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusOK, toRoomMemberResponse(member), i18n.MsgSuccess)
}

func (h *RoomHandler) JoinByInviteCode(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, i18n.ErrUnauthorized)
		return
	}

	var req dto.JoinRoomByInviteCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrValidation, err.Error())
		return
	}

	room, err := h.service.JoinByInviteCode(c.Request.Context(), userID, req.InviteCode)
	if err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusOK, toRoomResponse(room), i18n.MsgSuccess)
}

func toRoomResponse(r *model.Room) dto.RoomResponse {
	res := dto.RoomResponse{
		ID:          r.ID.String(),
		Type:        r.Type,
		Name:        r.Name,
		Description: r.Description,
		AvatarUrl:   r.AvatarUrl,
		CreatedBy:   r.CreatedBy.String(),
		InviteCode:  r.InviteCode,
		CreatedAt:   r.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   r.UpdatedAt.Format(time.RFC3339),
	}
	if r.LastMessageId != nil {
		s := r.LastMessageId.String()
		res.LastMessageID = &s
	}
	for _, m := range r.Members {
		m := m
		res.Members = append(res.Members, toRoomMemberResponse(&m))
	}
	return res
}

func toRoomMemberResponse(m *model.RoomMember) dto.RoomMemberResponse {
	res := dto.RoomMemberResponse{
		ID:         m.ID.String(),
		RoomID:     m.RoomId.String(),
		UserID:     m.UserId.String(),
		Role:       m.Role,
		Nickname:   m.Nickname,
		JoinedAt:   m.JoinedAt.Format(time.RFC3339),
		IsMuted:    m.IsMuted,
		IsArchived: m.IsArchived,
		IsPinned:   m.IsPinned,
		CreatedAt:  m.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  m.UpdatedAt.Format(time.RFC3339),
	}
	if m.LastReadMessageId != nil {
		s := m.LastReadMessageId.String()
		res.LastReadMessageID = &s
	}
	if m.LastReadAt != nil {
		t := m.LastReadAt.Format(time.RFC3339)
		res.LastReadAt = &t
	}
	return res
}
