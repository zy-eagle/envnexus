package agent

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/oklog/ulid/v2"

	"github.com/zy-eagle/envnexus/services/platform-api/internal/domain"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/dto"
	mw "github.com/zy-eagle/envnexus/services/platform-api/internal/middleware"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/repository"
)

type AuditHandler struct {
	auditRepo repository.AuditRepository
}

func NewAuditHandler(auditRepo repository.AuditRepository) *AuditHandler {
	return &AuditHandler{auditRepo: auditRepo}
}

func (h *AuditHandler) RegisterRoutes(router *gin.RouterGroup) {
	agentGroup := router.Group("/agent/v1")
	{
		agentGroup.POST("/audit-events", h.ReportEvents)
	}
}

func (h *AuditHandler) ReportEvents(c *gin.Context) {
	var req dto.AgentAuditEventsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		mw.RespondValidationError(c, err.Error())
		return
	}

	tenantID, _ := c.Get("tenant_id")
	tid, _ := tenantID.(string)
	deviceID, _ := c.Get("device_id")
	did, _ := deviceID.(string)

	if tid == "" {
		tid = c.GetHeader("X-Tenant-ID")
	}
	if did == "" {
		did = c.GetHeader("X-Device-ID")
	}

	var events []*domain.AuditEvent
	for _, item := range req.Events {
		payloadBytes, _ := json.Marshal(item.EventPayload)
		sessionID := item.SessionID
		var sessionPtr *string
		if sessionID != "" {
			sessionPtr = &sessionID
		}
		var devicePtr *string
		if did != "" {
			devicePtr = &did
		}
		events = append(events, &domain.AuditEvent{
			ID:               ulid.Make().String(),
			TenantID:         tid,
			DeviceID:         devicePtr,
			SessionID:        sessionPtr,
			EventType:        item.EventType,
			EventPayloadJSON: string(payloadBytes),
			CreatedAt:        time.Now(),
		})
	}

	if err := h.auditRepo.CreateBatch(c.Request.Context(), events); err != nil {
		mw.RespondError(c, domain.ErrInternalError)
		return
	}

	mw.RespondSuccess(c, http.StatusCreated, gin.H{
		"accepted": len(events),
	})
}
