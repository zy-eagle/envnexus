package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	mw "github.com/zy-eagle/envnexus/services/platform-api/internal/middleware"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/repository"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/audit"
)

type AuditHandler struct {
	auditService *audit.Service
}

func NewAuditHandler(auditService *audit.Service) *AuditHandler {
	return &AuditHandler{auditService: auditService}
}

func (h *AuditHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/tenants/:tenantId/audit-events", h.List)
}

func (h *AuditHandler) List(c *gin.Context) {
	tenantID := c.Param("tenantId")
	filters := repository.AuditFilters{
		DeviceID:  c.Query("device_id"),
		SessionID: c.Query("session_id"),
		EventType: c.Query("event_type"),
		StartAt:   c.Query("start_at"),
		EndAt:     c.Query("end_at"),
	}

	events, err := h.auditService.ListEvents(c.Request.Context(), tenantID, filters)
	if err != nil {
		mw.RespondError(c, err)
		return
	}
	mw.RespondSuccess(c, http.StatusOK, gin.H{"items": events})
}
