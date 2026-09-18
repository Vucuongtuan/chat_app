package response

import (
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"chatapp/pkg/i18n"
)

// ResponseBody represents the standard API response structure
type ResponseBody struct {
	Success    bool        `json:"success"`
	StatusCode int         `json:"status_code"`
	Message    string      `json:"message"`
	Data       interface{} `json:"data,omitempty"`
	Error      string      `json:"error,omitempty"`
	Timestamp  time.Time   `json:"timestamp"`
}

// Pagination represents pagination metadata
type Pagination struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// PaginatedResponseBody represents response structure for paginated lists
type PaginatedResponseBody struct {
	ResponseBody
	Pagination *Pagination `json:"pagination,omitempty"`
}

// Success sends a standardized success JSON response.
// messageKey can be an i18n key (e.g. i18n.MsgSuccess, i18n.MsgCreated) or custom text.
// If omitted, defaults to i18n.MsgSuccess.
func Success(c *gin.Context, statusCode int, data interface{}, messageKey ...string) {
	lang := i18n.GetLanguage(c)

	key := i18n.MsgSuccess
	if len(messageKey) > 0 && messageKey[0] != "" {
		key = messageKey[0]
	}

	msg := i18n.T(lang, key)

	c.JSON(statusCode, ResponseBody{
		Success:    true,
		StatusCode: statusCode,
		Message:    msg,
		Data:       data,
		Timestamp:  time.Now().UTC(),
	})
}

// Error sends a standardized error JSON response.
// errorCode represents standard error key (e.g. i18n.ErrInvalidFile, i18n.ErrBadRequest).
// If customMessage is provided, it overrides the translated message.
func Error(c *gin.Context, statusCode int, errorCode string, customMessage ...string) {
	lang := i18n.GetLanguage(c)

	var msg string
	if len(customMessage) > 0 && customMessage[0] != "" {
		msg = customMessage[0]
	} else {
		msg = i18n.T(lang, errorCode)
	}

	c.JSON(statusCode, ResponseBody{
		Success:    false,
		StatusCode: statusCode,
		Message:    msg,
		Error:      errorCode,
		Timestamp:  time.Now().UTC(),
	})
}

// Message sends a standardized message-only response without payload data.
func Message(c *gin.Context, statusCode int, messageKey string, args ...any) {
	lang := i18n.GetLanguage(c)
	msg := i18n.T(lang, messageKey, args...)

	isSuccess := statusCode >= 200 && statusCode < 300

	resp := ResponseBody{
		Success:    isSuccess,
		StatusCode: statusCode,
		Message:    msg,
		Timestamp:  time.Now().UTC(),
	}

	if !isSuccess {
		resp.Error = messageKey
	}

	c.JSON(statusCode, resp)
}

// Paginated sends a standardized paginated JSON response.
func Paginated(c *gin.Context, statusCode int, data interface{}, pagination *Pagination, messageKey ...string) {
	lang := i18n.GetLanguage(c)

	key := i18n.MsgSuccess
	if len(messageKey) > 0 && messageKey[0] != "" {
		key = messageKey[0]
	}

	msg := i18n.T(lang, key)

	c.JSON(statusCode, PaginatedResponseBody{
		ResponseBody: ResponseBody{
			Success:    true,
			StatusCode: statusCode,
			Message:    msg,
			Data:       data,
			Timestamp:  time.Now().UTC(),
		},
		Pagination: pagination,
	})
}

// CalculatePagination creates Pagination metadata based on page, perPage and total count
func CalculatePagination(page, perPage, total int) *Pagination {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}

	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(perPage)))
	}

	return &Pagination{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
	}
}

// BadRequest sends 400 Bad Request error
func BadRequest(c *gin.Context, errorCode string, customMessage ...string) {
	Error(c, http.StatusBadRequest, errorCode, customMessage...)
}

// Unauthorized sends 401 Unauthorized error
func Unauthorized(c *gin.Context, customMessage ...string) {
	Error(c, http.StatusUnauthorized, i18n.ErrUnauthorized, customMessage...)
}

// Forbidden sends 403 Forbidden error
func Forbidden(c *gin.Context, customMessage ...string) {
	Error(c, http.StatusForbidden, i18n.ErrForbidden, customMessage...)
}

// NotFound sends 404 Not Found error
func NotFound(c *gin.Context, customMessage ...string) {
	Error(c, http.StatusNotFound, i18n.ErrNotFound, customMessage...)
}

// InternalServerError sends 500 Internal Server Error
func InternalServerError(c *gin.Context, customMessage ...string) {
	Error(c, http.StatusInternalServerError, i18n.ErrInternalServer, customMessage...)
}
