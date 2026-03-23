package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/zy-eagle/envnexus/services/platform-api/internal/dto"
	mw "github.com/zy-eagle/envnexus/services/platform-api/internal/middleware"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/session"
)

type SessionHandler struct {
	sessionService *session.Service
}

func NewSessionHandler(sessionService *session.Service) *SessionHandler {
	return &SessionHandler{sessionService: sessionService}
}

func (h *SessionHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/tenants/:tenantId/sessions", h.List)
	router.POST("/sessions", h.Create)
	router.POST("/sessions/:sessionId/approve", h.Approve)
	router.POST("/sessions/:sessionId/deny", h.Deny)
	router.POST("/sessions/:sessionId/abort", h.Abort)
}

func (h *SessionHandler) List(c *gin.Context) {
	tenantID := c.Param("tenantId")
	sessions, err := h.sessionService.ListByTenant(c.Request.Context(), tenantID)
	if err != nil {
		mw.RespondError(c, err)
		return
	}
	mw.RespondSuccess(c, http.StatusOK, gin.H{"items": sessions})
}

func (h *SessionHandler) Create(c *gin.Context) {
	var req dto.CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		mw.RespondValidationError(c, err.Error())
		return
	}

	sess, err := h.sessionService.CreateSession(c.Request.Context(), req)
	if err != nil {
		mw.RespondError(c, err)
		return
	}
	mw.RespondSuccess(c, http.StatusCreated, gin.H{
		"session_id": sess.ID,
		"status":     sess.Status,
	})
}

func (h *SessionHandler) Approve(c *gin.Context) {
	sessionID := c.Param("sessionId")
	var req dto.ApproveSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		mw.RespondValidationError(c, err.Error())
		return
	}

	userID, _ := c.Get("user_id")
	approverID, _ := userID.(string)

	err := h.sessionService.ApproveSession(c.Request.Context(), sessionID, req.ApprovalRequestID, approverID, req.Comment)
	if err != nil {
		mw.RespondError(c, err)
		return
	}
	mw.RespondSuccess(c, http.StatusOK, gin.H{"status": "approved"})
}

func (h *SessionHandler) Deny(c *gin.Context) {
	sessionID := c.Param("sessionId")
	var req dto.DenySessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		mw.RespondValidationError(c, err.Error())
		return
	}

	err := h.sessionService.DenySession(c.Request.Context(), sessionID, req.ApprovalRequestID, req.Reason)
	if err != nil {
		mw.RespondError(c, err)
		return
	}
	mw.RespondSuccess(c, http.StatusOK, gin.H{"status": "denied"})
}

func (h *SessionHandler) Abort(c *gin.Context) {
	sessionID := c.Param("sessionId")
	var req dto.AbortSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		mw.RespondValidationError(c, err.Error())
		return
	}

	err := h.sessionService.AbortSession(c.Request.Context(), sessionID, req.Reason)
	if err != nil {
		mw.RespondError(c, err)
		return
	}
	mw.RespondSuccess(c, http.StatusOK, gin.H{"status": "aborted"})
}
