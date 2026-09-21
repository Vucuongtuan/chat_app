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

type MessageHandler struct {
	service *service.MessageService
}

func NewMessageHandler(s *service.MessageService) *MessageHandler {
	return &MessageHandler{service: s}
}

func (h *MessageHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware ...gin.HandlerFunc) {
	if len(authMiddleware) > 0 {
		router.Use(authMiddleware...)
	}
	// Routes gắn vào room
	router.POST("/rooms/:id/messages", h.SendMessage)
	router.GET("/rooms/:id/messages", h.ListMessages)
	router.POST("/rooms/:id/messages/read", h.MarkAsRead)

	// Routes theo message ID
	router.GET("/messages/:id", h.GetMessage)
	router.PUT("/messages/:id", h.EditMessage)
	router.DELETE("/messages/:id", h.DeleteMessage)
	router.POST("/messages/:id/reactions", h.AddReaction)
	router.DELETE("/messages/:id/reactions/:emoji", h.RemoveReaction)
}

func (h *MessageHandler) SendMessage(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, i18n.ErrUnauthorized)
		return
	}

	var req dto.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrValidation, err.Error())
		return
	}
	// Ưu tiên room_id từ URL param
	if roomID := c.Param("id"); roomID != "" {
		req.RoomID = roomID
	}

	msg, err := h.service.SendMessage(c.Request.Context(), userID, req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusCreated, toMessageResponse(msg), i18n.MsgSuccess)
}

func (h *MessageHandler) GetMessage(c *gin.Context) {
	userID := middleware.GetUserID(c)
	messageID := c.Param("id")

	msg, err := h.service.GetMessage(c.Request.Context(), userID, messageID)
	if err != nil {
		response.Error(c, http.StatusForbidden, i18n.ErrForbidden, err.Error())
		return
	}

	response.Success(c, http.StatusOK, toMessageResponse(msg), i18n.MsgSuccess)
}

func (h *MessageHandler) ListMessages(c *gin.Context) {
	userID := middleware.GetUserID(c)
	roomID := c.Param("id")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "30"))

	var beforeID *string
	if v := c.Query("before_id"); v != "" {
		beforeID = &v
	}

	msgs, total, err := h.service.ListMessages(c.Request.Context(), userID, roomID, page, perPage, beforeID)
	if err != nil {
		response.Error(c, http.StatusForbidden, i18n.ErrForbidden, err.Error())
		return
	}

	result := make([]dto.MessageResponse, len(msgs))
	for i := range msgs {
		result[i] = toMessageResponse(&msgs[i])
	}

	response.Success(c, http.StatusOK, gin.H{"items": result, "total": total, "page": page, "per_page": perPage}, i18n.MsgSuccess)
}

func (h *MessageHandler) EditMessage(c *gin.Context) {
	userID := middleware.GetUserID(c)
	messageID := c.Param("id")

	var req dto.EditMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrValidation, err.Error())
		return
	}

	msg, err := h.service.EditMessage(c.Request.Context(), userID, messageID, req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusOK, toMessageResponse(msg), i18n.MsgSuccess)
}

func (h *MessageHandler) DeleteMessage(c *gin.Context) {
	userID := middleware.GetUserID(c)
	messageID := c.Param("id")

	if err := h.service.DeleteMessage(c.Request.Context(), userID, messageID); err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrBadRequest, err.Error())
		return
	}

	response.Message(c, http.StatusOK, i18n.MsgSuccess)
}

func (h *MessageHandler) AddReaction(c *gin.Context) {
	userID := middleware.GetUserID(c)
	messageID := c.Param("id")

	var req dto.AddReactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrValidation, err.Error())
		return
	}

	reaction, err := h.service.AddReaction(c.Request.Context(), userID, messageID, req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusCreated, toReactionResponse(reaction), i18n.MsgSuccess)
}

func (h *MessageHandler) RemoveReaction(c *gin.Context) {
	userID := middleware.GetUserID(c)
	messageID := c.Param("id")
	emoji := c.Param("emoji")

	if err := h.service.RemoveReaction(c.Request.Context(), userID, messageID, emoji); err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrBadRequest, err.Error())
		return
	}

	response.Message(c, http.StatusOK, i18n.MsgSuccess)
}

func (h *MessageHandler) MarkAsRead(c *gin.Context) {
	userID := middleware.GetUserID(c)
	roomID := c.Param("id")

	var body struct {
		MessageID string `json:"message_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrValidation, err.Error())
		return
	}

	if err := h.service.MarkAsRead(c.Request.Context(), userID, roomID, body.MessageID); err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrBadRequest, err.Error())
		return
	}

	response.Message(c, http.StatusOK, i18n.MsgSuccess)
}

func toMessageResponse(m *model.Message) dto.MessageResponse {
	res := dto.MessageResponse{
		ID:        m.ID.String(),
		RoomID:    m.RoomId.String(),
		SenderID:  m.SenderId.String(),
		Type:      m.Type,
		Content:   m.Content,
		IsEdited:  m.IsEdited,
		IsDeleted: m.IsDeleted,
		CreatedAt: m.CreatedAt.Format(time.RFC3339),
		UpdatedAt: m.UpdatedAt.Format(time.RFC3339),
	}
	if m.ReplyToId != nil {
		s := m.ReplyToId.String()
		res.ReplyToID = &s
	}
	if m.ForwardedFromId != nil {
		s := m.ForwardedFromId.String()
		res.ForwardedFromID = &s
	}
	if m.EditedAt != nil {
		t := m.EditedAt.Format(time.RFC3339)
		res.EditedAt = &t
	}
	if m.DeletedAt != nil {
		t := m.DeletedAt.Format(time.RFC3339)
		res.DeletedAt = &t
	}
	for _, r := range m.Reactions {
		r := r
		res.Reactions = append(res.Reactions, toReactionResponse(&r))
	}
	return res
}

func toReactionResponse(r *model.MessageReaction) dto.MessageReactionResponse {
	return dto.MessageReactionResponse{
		ID:        r.ID.String(),
		MessageID: r.MessageId.String(),
		UserID:    r.UserId.String(),
		Emoji:     r.Emoji,
		CreatedAt: r.CreatedAt.Format(time.RFC3339),
		UpdatedAt: r.UpdatedAt.Format(time.RFC3339),
	}
}
