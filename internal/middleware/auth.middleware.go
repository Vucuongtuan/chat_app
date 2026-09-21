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
	ContextUserID    = "user_id"
	ContextAccountID = "account_id"
	ContextSessionID = "session_id"
	ContextDeviceID  = "device_id"
	ContextIsPrimary = "is_primary"
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
		c.Set(ContextAccountID, claims.AccountID)
		c.Set(ContextSessionID, claims.SessionID)
		c.Set(ContextDeviceID, claims.DeviceID)
		c.Set(ContextIsPrimary, claims.IsPrimary)
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

func GetAccountID(c *gin.Context) string {
	if val, exists := c.Get(ContextAccountID); exists {
		if id, ok := val.(string); ok {
			return id
		}
	}
	return ""
}

func GetSessionID(c *gin.Context) string {
	if val, exists := c.Get(ContextSessionID); exists {
		if id, ok := val.(string); ok {
			return id
		}
	}
	return ""
}

func GetDeviceID(c *gin.Context) string {
	if val, exists := c.Get(ContextDeviceID); exists {
		if id, ok := val.(string); ok {
			return id
		}
	}
	return ""
}

func IsPrimaryDevice(c *gin.Context) bool {
	if val, exists := c.Get(ContextIsPrimary); exists {
		if isPrimary, ok := val.(bool); ok {
			return isPrimary
		}
	}
	return false
}

// RequirePrimaryDevice verifies that the incoming request is made from the primary device
func RequirePrimaryDevice() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !IsPrimaryDevice(c) {
			response.Error(c, http.StatusForbidden, i18n.ErrOnlyPrimaryDeviceAllowed)
			c.Abort()
			return
		}
		c.Next()
	}
}
