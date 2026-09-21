package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"chatapp/pkg/i18n"
	"chatapp/pkg/response"
)

type Handlers struct {
	Media   *MediaHandler
	User    *UserHandler
	Auth    *AuthHandler
	Room    *RoomHandler
	Message *MessageHandler
	Post    *PostHandler
}

func Setup(router *gin.Engine, h *Handlers, authMiddleware ...gin.HandlerFunc) {
	api := router.Group("/api/v1")
	{
		api.GET("/test", func(c *gin.Context) {
			response.Success(c, http.StatusOK, gin.H{"status": "ok"}, i18n.MsgSuccess)
		})

		if h.Auth != nil {
			h.Auth.RegisterRoutes(api, authMiddleware...)
		}

		if h.Media != nil {
			h.Media.RegisterRoutes(api)
		}

		if h.User != nil {
			h.User.RegisterRoutes(api, authMiddleware...)
		}

		if h.Room != nil {
			h.Room.RegisterRoutes(api, authMiddleware...)
		}

		if h.Message != nil {
			h.Message.RegisterRoutes(api, authMiddleware...)
		}

		if h.Post != nil {
			h.Post.RegisterRoutes(api, authMiddleware...)
		}
	}
}
