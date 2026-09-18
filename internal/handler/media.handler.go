package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"chatapp/internal/service"
	"chatapp/pkg/i18n"
	"chatapp/pkg/response"
)

func (h *MediaHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/media/upload", h.Upload)
	router.GET("/media", h.GetByOwner)
	router.DELETE("/media/:id", h.Delete)
}

type MediaHandler struct {
	service *service.MediaService
}

func NewMediaHandler(s *service.MediaService) *MediaHandler {
	return &MediaHandler{service: s}
}

func (h *MediaHandler) Upload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrInvalidFile)
		return
	}

	ownerType := c.PostForm("owner_type")
	if ownerType == "" {
		response.Error(c, http.StatusBadRequest, i18n.ErrMissingOwnerType)
		return
	}

	ownerIdStr := c.PostForm("owner_id")
	ownerId, err := uuid.Parse(ownerIdStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrInvalidOwnerID)
		return
	}

	sortOrder, _ := strconv.Atoi(c.DefaultPostForm("sort_order", "0"))

	media, err := h.service.Upload(c.Request.Context(), file, ownerType, ownerId, sortOrder)
	if err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusCreated, media, i18n.MsgMediaUploaded)
}

func (h *MediaHandler) GetByOwner(c *gin.Context) {
	ownerType := c.Query("owner_type")
	ownerIdStr := c.Query("owner_id")

	ownerId, err := uuid.Parse(ownerIdStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrInvalidOwnerID)
		return
	}

	media, err := h.service.GetByOwner(c.Request.Context(), ownerType, ownerId)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, i18n.ErrServer)
		return
	}

	response.Success(c, http.StatusOK, media, i18n.MsgSuccess)
}

func (h *MediaHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrInvalidID)
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, i18n.ErrDeleteFailed)
		return
	}

	response.Message(c, http.StatusOK, i18n.MsgMediaDeleted)
}
