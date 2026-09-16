package handler

import "github.com/gin-gonic/gin"

type Handlers struct {
	Media *MediaHandler
}

func Setup(router *gin.Engine, h *Handlers) {
	api := router.Group("/api/v1")
	{
		api.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "API is working",
			})
		})

		h.Media.RegisterRoutes(api)
	}
}
