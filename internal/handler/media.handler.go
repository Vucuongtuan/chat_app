package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"chatapp/internal/service"
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "file không hợp lệ"})
		return
	}

	ownerType := c.PostForm("owner_type")
	if ownerType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "thiếu owner_type"})
		return
	}

	ownerIdStr := c.PostForm("owner_id")
	ownerId, err := strconv.ParseUint(ownerIdStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "owner_id không hợp lệ"})
		return
	}

	sortOrder, _ := strconv.Atoi(c.DefaultPostForm("sort_order", "0"))

	media, err := h.service.Upload(file, ownerType, uint(ownerId), sortOrder)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, media)
}

func (h *MediaHandler) GetByOwner(c *gin.Context) {
	ownerType := c.Query("owner_type")
	ownerIdStr := c.Query("owner_id")

	ownerId, err := strconv.ParseUint(ownerIdStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "owner_id không hợp lệ"})
		return
	}

	media, err := h.service.GetByOwner(ownerType, uint(ownerId))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "lỗi server"})
		return
	}

	c.JSON(http.StatusOK, media)
}

func (h *MediaHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id không hợp lệ"})
		return
	}

	if err := h.service.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "xóa thất bại"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "đã xóa"})
}
