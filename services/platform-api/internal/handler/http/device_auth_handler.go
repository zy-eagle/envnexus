package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/zy-eagle/envnexus/services/platform-api/internal/dto"
	mw "github.com/zy-eagle/envnexus/services/platform-api/internal/middleware"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/device_auth"
)

// DeviceAuthHandler exposes RFC 8628-style device login and IDE token refresh.
type DeviceAuthHandler struct {
	svc *device_auth.Service
}

func NewDeviceAuthHandler(svc *device_auth.Service) *DeviceAuthHandler {
	return &DeviceAuthHandler{svc: svc}
}

// RegisterPublicRoutes device init, poll, and refresh (no user session).
func (h *DeviceAuthHandler) RegisterPublicRoutes(router *gin.RouterGroup) {
	router.POST("/device-auth/init", h.Init)
	router.POST("/device-auth/poll", h.Poll)
	router.POST("/device-auth/refresh", h.Refresh)
}

// RegisterProtectedRoutes require JWT (confirm flow).
func (h *DeviceAuthHandler) RegisterProtectedRoutes(router *gin.RouterGroup) {
	router.POST("/device-auth/confirm", h.Confirm)
}

func (h *DeviceAuthHandler) Init(c *gin.Context) {
	var req dto.DeviceAuthInitAPIRequest
	_ = c.ShouldBindJSON(&req)
	out, err := h.svc.InitDeviceAuthorization(c.Request.Context(), req.DeviceInfo)
	if err != nil {
		mw.RespondError(c, err)
		return
	}
	mw.RespondSuccess(c, http.StatusOK, out)
}

func (h *DeviceAuthHandler) Poll(c *gin.Context) {
	var req dto.DeviceAuthPollAPIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		mw.RespondValidationError(c, err.Error())
		return
	}
	out, err := h.svc.PollForTokens(c.Request.Context(), req.DeviceCode)
	if err != nil {
		mw.RespondError(c, err)
		return
	}
	// 200 for RFC 8628-style errors: authorization_pending, access_denied, expired_token, or tokens.
	mw.RespondSuccess(c, http.StatusOK, out)
}

func (h *DeviceAuthHandler) Refresh(c *gin.Context) {
	var req dto.IdeClientRefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		mw.RespondValidationError(c, err.Error())
		return
	}
	out, err := h.svc.RefreshTokens(c.Request.Context(), req.RefreshToken)
	if err != nil {
		mw.RespondError(c, err)
		return
	}
	mw.RespondSuccess(c, http.StatusOK, out)
}

func (h *DeviceAuthHandler) Confirm(c *gin.Context) {
	var req dto.DeviceAuthConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		mw.RespondValidationError(c, err.Error())
		return
	}
	userID, ok := c.Get("user_id")
	if !ok {
		mw.RespondError(c, mw.ErrUnauthorizedFromContext())
		return
	}
	tenantID, ok := c.Get("tenant_id")
	if !ok {
		mw.RespondError(c, mw.ErrUnauthorizedFromContext())
		return
	}
	uid, ok := userID.(string)
	if !ok || uid == "" {
		mw.RespondError(c, mw.ErrUnauthorizedFromContext())
		return
	}
	tid, ok := tenantID.(string)
	if !ok || tid == "" {
		mw.RespondError(c, mw.ErrUnauthorizedFromContext())
		return
	}
	if err := h.svc.ConfirmUserCode(c.Request.Context(), req.UserCode, uid, tid, req.Approve); err != nil {
		mw.RespondError(c, err)
		return
	}
	mw.RespondSuccess(c, http.StatusOK, gin.H{"status": "ok"})
}
