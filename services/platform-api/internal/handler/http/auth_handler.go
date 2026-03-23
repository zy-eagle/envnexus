package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/zy-eagle/envnexus/services/platform-api/internal/dto"
	mw "github.com/zy-eagle/envnexus/services/platform-api/internal/middleware"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/auth"
)

type AuthHandler struct {
	authService *auth.Service
}

func NewAuthHandler(authService *auth.Service) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) RegisterRoutes(router *gin.RouterGroup) {
	authGroup := router.Group("/auth")
	{
		authGroup.POST("/login", h.Login)
	}
	router.GET("/me", h.Me)
	router.GET("/bootstrap", h.Bootstrap)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		mw.RespondValidationError(c, err.Error())
		return
	}

	resp, err := h.authService.Login(c.Request.Context(), req)
	if err != nil {
		mw.RespondError(c, err)
		return
	}
	mw.RespondSuccess(c, http.StatusOK, resp)
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		mw.RespondError(c, mw.ErrUnauthorizedFromContext())
		return
	}

	user, err := h.authService.GetUserByID(c.Request.Context(), userID.(string))
	if err != nil || user == nil {
		mw.RespondError(c, mw.ErrUnauthorizedFromContext())
		return
	}

	tenantID, _ := c.Get("tenant_id")
	mw.RespondSuccess(c, http.StatusOK, gin.H{
		"user": gin.H{
			"id":           user.ID,
			"tenant_id":    user.TenantID,
			"email":        user.Email,
			"display_name": user.DisplayName,
			"status":       user.Status,
		},
		"tenant_id": tenantID,
	})
}

func (h *AuthHandler) Bootstrap(c *gin.Context) {
	mw.RespondSuccess(c, http.StatusOK, gin.H{
		"platform":    "EnvNexus",
		"version":     "0.1.0",
		"api_version": "v1",
		"features": gin.H{
			"multi_tenant":    true,
			"websocket":       true,
			"webhook":         false,
			"repair_tools":    true,
			"diagnosis_tools": true,
		},
	})
}
