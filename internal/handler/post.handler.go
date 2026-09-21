package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"chatapp/internal/dto"
	"chatapp/internal/middleware"
	"chatapp/internal/service"
	"chatapp/pkg/i18n"
	"chatapp/pkg/response"
)

type PostHandler struct {
	service *service.PostService
}

func NewPostHandler(s *service.PostService) *PostHandler {
	return &PostHandler{service: s}
}

func (h *PostHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware ...gin.HandlerFunc) {
	posts := router.Group("/posts")
	{
		// Public
		posts.GET("", h.ListPosts)
		posts.GET("/:id", h.GetPost)
		posts.GET("/:id/comments", h.ListComments)
	}

	if len(authMiddleware) > 0 {
		protected := posts.Group("")
		protected.Use(authMiddleware...)
		{
			protected.POST("", h.CreatePost)
			protected.PUT("/:id", h.UpdatePost)
			protected.DELETE("/:id", h.DeletePost)
			protected.POST("/:id/like", h.ToggleLike)
			protected.POST("/:id/comments", h.AddComment)
			protected.DELETE("/:id/comments/:commentId", h.DeleteComment)
		}

		// Lấy danh sách bài viết của một user cụ thể
		router.GET("/users/:id/posts", h.ListUserPosts)
	}
}

func (h *PostHandler) CreatePost(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, i18n.ErrUnauthorized)
		return
	}

	var req dto.CreatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrValidation, err.Error())
		return
	}

	post, err := h.service.CreatePost(c.Request.Context(), userID, req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusCreated, post, i18n.MsgSuccess)
}

func (h *PostHandler) GetPost(c *gin.Context) {
	postID := c.Param("id")
	currentUserID := middleware.GetUserID(c)

	post, err := h.service.GetPost(c.Request.Context(), currentUserID, postID)
	if err != nil {
		response.Error(c, http.StatusNotFound, i18n.ErrNotFound, err.Error())
		return
	}

	response.Success(c, http.StatusOK, post, i18n.MsgSuccess)
}

func (h *PostHandler) ListPosts(c *gin.Context) {
	currentUserID := middleware.GetUserID(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	posts, total, err := h.service.ListPosts(c.Request.Context(), currentUserID, page, perPage)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, i18n.ErrServer, err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"items": posts, "total": total, "page": page, "per_page": perPage}, i18n.MsgSuccess)
}

func (h *PostHandler) ListUserPosts(c *gin.Context) {
	targetUserID := c.Param("id")
	currentUserID := middleware.GetUserID(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	posts, total, err := h.service.ListUserPosts(c.Request.Context(), currentUserID, targetUserID, page, perPage)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, i18n.ErrServer, err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"items": posts, "total": total, "page": page, "per_page": perPage}, i18n.MsgSuccess)
}

func (h *PostHandler) UpdatePost(c *gin.Context) {
	userID := middleware.GetUserID(c)
	postID := c.Param("id")

	var req dto.UpdatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrValidation, err.Error())
		return
	}

	post, err := h.service.UpdatePost(c.Request.Context(), userID, postID, req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusOK, post, i18n.MsgSuccess)
}

func (h *PostHandler) DeletePost(c *gin.Context) {
	userID := middleware.GetUserID(c)
	postID := c.Param("id")

	if err := h.service.DeletePost(c.Request.Context(), userID, postID); err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrBadRequest, err.Error())
		return
	}

	response.Message(c, http.StatusOK, i18n.MsgSuccess)
}

func (h *PostHandler) ToggleLike(c *gin.Context) {
	userID := middleware.GetUserID(c)
	postID := c.Param("id")

	isLiked, err := h.service.ToggleLike(c.Request.Context(), userID, postID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"is_liked": isLiked}, i18n.MsgSuccess)
}

func (h *PostHandler) AddComment(c *gin.Context) {
	userID := middleware.GetUserID(c)
	postID := c.Param("id")

	var req dto.CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrValidation, err.Error())
		return
	}

	comment, err := h.service.AddComment(c.Request.Context(), userID, postID, req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusCreated, comment, i18n.MsgSuccess)
}

func (h *PostHandler) ListComments(c *gin.Context) {
	postID := c.Param("id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	comments, total, err := h.service.ListComments(c.Request.Context(), postID, page, perPage)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, i18n.ErrServer, err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"items": comments, "total": total, "page": page, "per_page": perPage}, i18n.MsgSuccess)
}

func (h *PostHandler) DeleteComment(c *gin.Context) {
	userID := middleware.GetUserID(c)
	commentID := c.Param("commentId")

	if err := h.service.DeleteComment(c.Request.Context(), userID, commentID); err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrBadRequest, err.Error())
		return
	}

	response.Message(c, http.StatusOK, i18n.MsgSuccess)
}
