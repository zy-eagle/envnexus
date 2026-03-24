package http

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/zy-eagle/envnexus/services/platform-api/internal/dto"
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
	router.GET("/tenants/:tenantId/audit-events/export", h.Export)
}

func (h *AuditHandler) List(c *gin.Context) {
	tenantID := c.Param("tenantId")
	filters := repository.AuditFilters{
		DeviceID:        c.Query("device_id"),
		SessionID:       c.Query("session_id"),
		EventType:       c.Query("event_type"),
		StartAt:         c.Query("start_at"),
		EndAt:           c.Query("end_at"),
		IncludeArchived: c.Query("include_archived") == "true",
	}

	events, err := h.auditService.ListEvents(c.Request.Context(), tenantID, filters)
	if err != nil {
		mw.RespondError(c, err)
		return
	}
	mw.RespondSuccess(c, http.StatusOK, gin.H{"items": events})
}

// Export returns audit events as NDJSON with PII redacted.
func (h *AuditHandler) Export(c *gin.Context) {
	tenantID := c.Param("tenantId")
	filters := repository.AuditFilters{
		DeviceID:        c.Query("device_id"),
		SessionID:       c.Query("session_id"),
		EventType:       c.Query("event_type"),
		StartAt:         c.Query("start_at"),
		EndAt:           c.Query("end_at"),
		IncludeArchived: c.Query("include_archived") == "true",
	}

	events, err := h.auditService.ListEvents(c.Request.Context(), tenantID, filters)
	if err != nil {
		mw.RespondError(c, err)
		return
	}

	redact := c.Query("redact_pii") != "false"

	c.Header("Content-Type", "application/x-ndjson")
	c.Header("Content-Disposition", "attachment; filename=audit-export-"+tenantID+".ndjson")

	encoder := json.NewEncoder(c.Writer)
	for _, evt := range events {
		exported := evt
		if redact {
			exported = redactAuditEvent(exported)
		}
		_ = encoder.Encode(exported)
	}
}

var (
	emailPattern = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)
	ipPattern    = regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`)
	tokenPattern = regexp.MustCompile(`(eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,})`)
)

func redactAuditEvent(evt *dto.AuditEventResponse) *dto.AuditEventResponse {
	cp := *evt
	if cp.EventPayloadJSON != "" {
		cp.EventPayloadJSON = redactString(cp.EventPayloadJSON)
	}
	return &cp
}

func redactString(s string) string {
	s = emailPattern.ReplaceAllStringFunc(s, func(match string) string {
		parts := strings.SplitN(match, "@", 2)
		if len(parts) != 2 {
			return "***@***"
		}
		local := parts[0]
		if len(local) > 2 {
			local = string(local[0]) + "***" + string(local[len(local)-1])
		} else {
			local = "***"
		}
		return local + "@" + parts[1]
	})
	s = tokenPattern.ReplaceAllString(s, "***JWT_REDACTED***")
	s = ipPattern.ReplaceAllStringFunc(s, func(match string) string {
		parts := strings.SplitN(match, ".", 4)
		if len(parts) == 4 {
			return parts[0] + "." + parts[1] + ".xxx.xxx"
		}
		return "xxx.xxx.xxx.xxx"
	})
	return s
}
