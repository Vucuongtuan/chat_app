package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"chatapp/pkg/i18n"
	"chatapp/pkg/jwt"
	"chatapp/pkg/response"
)

const (
	ContextUserID = "user_id"
)

func Auth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Error(c, http.StatusUnauthorized, i18n.ErrUnauthorized)
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && strings.ToLower(parts[0]) == "bearer") {
			response.Error(c, http.StatusUnauthorized, i18n.ErrUnauthorized)
			c.Abort()
			return
		}

		claims, err := jwt.ValidateToken(parts[1], secret)
		if err != nil {
			response.Error(c, http.StatusUnauthorized, i18n.ErrUnauthorized)
			c.Abort()
			return
		}

		c.Set(ContextUserID, claims.UserID)
		c.Next()
	}
}

func GetUserID(c *gin.Context) string {
	if val, exists := c.Get(ContextUserID); exists {
		if id, ok := val.(string); ok {
			return id
		}
	}
	return ""
}
