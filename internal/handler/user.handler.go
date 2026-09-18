package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"chatapp/internal/dto"
	"chatapp/internal/middleware"
	"chatapp/internal/model"
	"chatapp/internal/service"
	"chatapp/pkg/i18n"
	"chatapp/pkg/response"
)

type UserHandler struct {
	service *service.UserService
}

func NewUserHandler(s *service.UserService) *UserHandler {
	return &UserHandler{service: s}
}

func (h *UserHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware ...gin.HandlerFunc) {
	users := router.Group("/users")

	// Protected routes (require auth if provided)
	if len(authMiddleware) > 0 {
		protected := users.Group("")
		protected.Use(authMiddleware...)
		{
			protected.GET("/me", h.GetMe)
			protected.PUT("/profile", h.UpdateMe)
		}
	}

	// Public routes
	users.GET("", h.Find)
	users.GET("/:id", h.FindByID)
	users.PUT("/:id", h.UpdateByID)
}

// FindByID retrieves a user profile by UUID
func (h *UserHandler) FindByID(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrInvalidUserID)
		return
	}

	user, err := h.service.FindByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, i18n.ErrUserNotFound)
			return
		}
		response.Error(c, http.StatusInternalServerError, i18n.ErrServer)
		return
	}

	response.Success(c, http.StatusOK, toUserResponse(user), i18n.MsgSuccess)
}

// Find searches or finds users by query parameters (?display_name=... or ?q=...)
func (h *UserHandler) Find(c *gin.Context) {
	displayName := strings.TrimSpace(c.Query("display_name"))
	if displayName != "" {
		user, err := h.service.FindByDisplayName(c.Request.Context(), displayName)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				response.Error(c, http.StatusNotFound, i18n.ErrUserNotFound)
				return
			}
			response.Error(c, http.StatusInternalServerError, i18n.ErrServer)
			return
		}
		response.Success(c, http.StatusOK, toUserResponse(user), i18n.MsgSuccess)
		return
	}

	q := strings.TrimSpace(c.Query("q"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	users, err := h.service.Search(c.Request.Context(), q, limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, i18n.ErrServer)
		return
	}

	response.Success(c, http.StatusOK, toUserResponseList(users), i18n.MsgSuccess)
}

// GetMe retrieves the authenticated user's profile
func (h *UserHandler) GetMe(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, i18n.ErrUnauthorized)
		return
	}

	user, err := h.service.FindByID(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, i18n.ErrUserNotFound)
			return
		}
		response.Error(c, http.StatusInternalServerError, i18n.ErrServer)
		return
	}

	response.Success(c, http.StatusOK, toUserResponse(user), i18n.MsgSuccess)
}

// UpdateMe updates profile for the authenticated user
func (h *UserHandler) UpdateMe(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, i18n.ErrUnauthorized)
		return
	}

	var req dto.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrValidation, err.Error())
		return
	}

	user, err := h.service.UpdateProfile(c.Request.Context(), userID, req)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, i18n.ErrUserNotFound)
			return
		}
		response.Error(c, http.StatusBadRequest, i18n.ErrBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusOK, toUserResponse(user), i18n.MsgProfileUpdated)
}

// UpdateByID updates profile for a user by id in param
func (h *UserHandler) UpdateByID(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrInvalidUserID)
		return
	}

	// If request has authenticated user, ensure they can only update their own profile
	currentUserID := middleware.GetUserID(c)
	if currentUserID != "" && currentUserID != id {
		response.Error(c, http.StatusForbidden, i18n.ErrForbidden)
		return
	}

	var req dto.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrValidation, err.Error())
		return
	}

	user, err := h.service.UpdateProfile(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, i18n.ErrUserNotFound)
			return
		}
		response.Error(c, http.StatusBadRequest, i18n.ErrBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusOK, toUserResponse(user), i18n.MsgProfileUpdated)
}

func toUserResponse(u *model.User) dto.UserResponse {
	res := dto.UserResponse{
		ID:          u.ID.String(),
		FullName:    u.FullName,
		DisplayName: u.DisplayName,
		AvatarUrl:   u.AvatarUrl,
		CoverUrl:    u.CoverUrl,
		Bio:         u.Bio,
		Status:      u.Status,
		CreatedAt:   u.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   u.UpdatedAt.Format(time.RFC3339),
	}
	if u.LastLoginAt != nil {
		t := u.LastLoginAt.Format(time.RFC3339)
		res.LastLoginAt = &t
	}
	return res
}

func toUserResponseList(users []model.User) []dto.UserResponse {
	res := make([]dto.UserResponse, len(users))
	for i := range users {
		res[i] = toUserResponse(&users[i])
	}
	return res
}
