package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"chatapp/internal/dto"
	"chatapp/internal/middleware"
	"chatapp/internal/service"
	"chatapp/pkg/i18n"
	"chatapp/pkg/response"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware ...gin.HandlerFunc) {
	auth := router.Group("/auth")
	{
		// Public routes
		auth.POST("/register", h.Register)
		auth.POST("/login", h.Login)
		auth.POST("/2fa/verify", h.VerifyLogin2FA)
		auth.POST("/forgot-password", h.ForgotPassword)
		auth.POST("/reset-password", h.ResetPassword)
		auth.POST("/refresh-token", h.RefreshToken)

		// Protected routes
		if len(authMiddleware) > 0 {
			protected := auth.Group("")
			protected.Use(authMiddleware...)
			{
				protected.POST("/logout", h.Logout)
				protected.GET("/devices", h.GetDevices)

				// Chỉ thiết bị chính (Primary Device) mới có quyền thao tác các routes sau
				primaryOnly := protected.Group("")
				primaryOnly.Use(middleware.RequirePrimaryDevice())
				{
					// Quản lý thiết bị từ xa
					primaryOnly.DELETE("/devices/:id", h.LogoutDevice)
					primaryOnly.POST("/devices/logout-others", h.LogoutAllOtherDevices)
					primaryOnly.POST("/devices/:id/set-primary", h.TransferPrimary)

					// Bật / tắt 2FA qua email
					primaryOnly.POST("/2fa/enable-request", h.RequestEnable2FA)
					primaryOnly.POST("/2fa/enable-confirm", h.ConfirmEnable2FA)
					primaryOnly.POST("/2fa/disable", h.Disable2FA)
				}
			}
		}
	}
}

// Register
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrValidation, err.Error())
		return
	}

	ipAddress := c.ClientIP()
	userAgent := c.Request.UserAgent()

	res, err := h.authService.Register(c.Request.Context(), req, ipAddress, userAgent)
	if err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusCreated, res, i18n.MsgRegisterSuccess)
}

// Login
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrValidation, err.Error())
		return
	}

	ipAddress := c.ClientIP()
	userAgent := c.Request.UserAgent()

	res, err := h.authService.Login(c.Request.Context(), req, ipAddress, userAgent)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, i18n.ErrInvalidCredentials, err.Error())
		return
	}

	if res.TwoFactorRequired {
		response.Success(c, http.StatusOK, res, i18n.ErrTwoFactorRequired)
		return
	}

	response.Success(c, http.StatusOK, res, i18n.MsgLoginSuccess)
}

// VerifyLogin2FA
func (h *AuthHandler) VerifyLogin2FA(c *gin.Context) {
	var req dto.Verify2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrValidation, err.Error())
		return
	}

	ipAddress := c.ClientIP()
	userAgent := c.Request.UserAgent()

	res, err := h.authService.VerifyLogin2FA(c.Request.Context(), req, ipAddress, userAgent)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, i18n.ErrInvalidOTP, err.Error())
		return
	}

	response.Success(c, http.StatusOK, res, i18n.MsgLoginSuccess)
}

// RefreshToken
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrValidation, err.Error())
		return
	}

	res, err := h.authService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, i18n.ErrUnauthorized, err.Error())
		return
	}

	response.Success(c, http.StatusOK, res, i18n.MsgSuccess)
}

// Logout
func (h *AuthHandler) Logout(c *gin.Context) {
	sessionID := middleware.GetSessionID(c)
	if sessionID == "" {
		response.Error(c, http.StatusBadRequest, i18n.ErrSessionNotFound)
		return
	}

	if err := h.authService.Logout(c.Request.Context(), sessionID); err != nil {
		response.Error(c, http.StatusInternalServerError, i18n.ErrServer, err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"status": "logged_out"}, i18n.MsgSuccess)
}

// ForgotPassword
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req dto.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrValidation, err.Error())
		return
	}

	if err := h.authService.ForgotPassword(c.Request.Context(), req.Email); err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"status": "otp_sent"}, i18n.MsgOTPSent)
}

// ResetPassword
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req dto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrValidation, err.Error())
		return
	}

	if err := h.authService.ResetPassword(c.Request.Context(), req); err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"status": "password_reset"}, i18n.MsgPasswordResetSuccess)
}

// GetDevices
func (h *AuthHandler) GetDevices(c *gin.Context) {
	accountID := middleware.GetAccountID(c)
	sessionID := middleware.GetSessionID(c)

	devices, err := h.authService.GetDevices(c.Request.Context(), accountID, sessionID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, i18n.ErrServer, err.Error())
		return
	}

	response.Success(c, http.StatusOK, devices, i18n.MsgSuccess)
}

// LogoutDevice (Chỉ Primary Device)
func (h *AuthHandler) LogoutDevice(c *gin.Context) {
	targetSessionID := c.Param("id")
	accountID := middleware.GetAccountID(c)
	currentSessionID := middleware.GetSessionID(c)
	isPrimary := middleware.IsPrimaryDevice(c)

	err := h.authService.LogoutDevice(c.Request.Context(), accountID, currentSessionID, targetSessionID, isPrimary)
	if err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"revoked_session_id": targetSessionID}, i18n.MsgDeviceLoggedOut)
}

// LogoutAllOtherDevices (Chỉ Primary Device)
func (h *AuthHandler) LogoutAllOtherDevices(c *gin.Context) {
	accountID := middleware.GetAccountID(c)
	currentSessionID := middleware.GetSessionID(c)
	isPrimary := middleware.IsPrimaryDevice(c)

	err := h.authService.LogoutAllOtherDevices(c.Request.Context(), accountID, currentSessionID, isPrimary)
	if err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"status": "all_other_devices_logged_out"}, i18n.MsgAllOtherDevicesLoggedOut)
}

// TransferPrimary (Chuyển quyền thiết bị chính)
func (h *AuthHandler) TransferPrimary(c *gin.Context) {
	targetSessionID := c.Param("id")
	accountID := middleware.GetAccountID(c)
	isPrimary := middleware.IsPrimaryDevice(c)

	var req struct {
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrValidation, err.Error())
		return
	}

	err := h.authService.TransferPrimary(c.Request.Context(), accountID, targetSessionID, req.Password, isPrimary)
	if err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"new_primary_session_id": targetSessionID}, i18n.MsgPrimaryTransferred)
}

// RequestEnable2FA (Chỉ Primary Device: gửi OTP đến email)
func (h *AuthHandler) RequestEnable2FA(c *gin.Context) {
	accountID := middleware.GetAccountID(c)
	isPrimary := middleware.IsPrimaryDevice(c)

	err := h.authService.RequestEnable2FA(c.Request.Context(), accountID, isPrimary)
	if err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"status": "otp_sent"}, i18n.MsgOTPSent)
}

// ConfirmEnable2FA (Chỉ Primary Device: nhập OTP để bật 2FA)
func (h *AuthHandler) ConfirmEnable2FA(c *gin.Context) {
	accountID := middleware.GetAccountID(c)
	isPrimary := middleware.IsPrimaryDevice(c)

	var req dto.Enable2FAConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrValidation, err.Error())
		return
	}

	err := h.authService.ConfirmEnable2FA(c.Request.Context(), accountID, strings.TrimSpace(req.OTPCode), isPrimary)
	if err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"two_factor_enabled": true}, i18n.Msg2FAEnabled)
}

// Disable2FA (Chỉ Primary Device: nhập password để tắt 2FA)
func (h *AuthHandler) Disable2FA(c *gin.Context) {
	accountID := middleware.GetAccountID(c)
	isPrimary := middleware.IsPrimaryDevice(c)

	var req dto.Disable2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrValidation, err.Error())
		return
	}

	err := h.authService.Disable2FA(c.Request.Context(), accountID, req.Password, isPrimary)
	if err != nil {
		response.Error(c, http.StatusBadRequest, i18n.ErrBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"two_factor_enabled": false}, i18n.Msg2FADisabled)
}
