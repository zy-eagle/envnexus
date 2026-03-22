package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/audit"
)

type AuditHandler struct {
	auditService *audit.Service
}

func NewAuditHandler(auditService *audit.Service) *AuditHandler {
	return &AuditHandler{
		auditService: auditService,
	}
}

func (h *AuditHandler) RegisterRoutes(router *gin.RouterGroup) {
	audits := router.Group("/tenants/:tenantId/audit-events")
	{
		audits.GET("", h.ListEvents)
	}
}

func (h *AuditHandler) ListEvents(c *gin.Context) {
	tenantID := c.Param("tenantId")
	resp, err := h.auditService.ListEvents(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": resp})
}
