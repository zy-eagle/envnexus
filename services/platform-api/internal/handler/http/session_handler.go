package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/session"
)

type SessionHandler struct {
	sessionService *session.Service
}

func NewSessionHandler(sessionService *session.Service) *SessionHandler {
	return &SessionHandler{
		sessionService: sessionService,
	}
}

func (h *SessionHandler) RegisterRoutes(router *gin.RouterGroup) {
	sessions := router.Group("/tenants/:tenantId/sessions")
	{
		sessions.GET("", h.ListSessions)
	}
}

func (h *SessionHandler) ListSessions(c *gin.Context) {
	tenantID := c.Param("tenantId")
	resp, err := h.sessionService.ListSessions(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": resp})
}
