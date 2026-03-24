package agent

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/zy-eagle/envnexus/services/platform-api/internal/domain"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/dto"
	mw "github.com/zy-eagle/envnexus/services/platform-api/internal/middleware"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/repository"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/device"
)

// LifecycleHandler handles agent heartbeat and config pull.
// Device identity is resolved by device.Service; profile lookups go
// through their respective repositories (no dedicated profile service exists yet).
type LifecycleHandler struct {
	deviceService     *device.Service
	agentProfileRepo  repository.AgentProfileRepository
	modelProfileRepo  repository.ModelProfileRepository
	policyProfileRepo repository.PolicyProfileRepository
}

func NewLifecycleHandler(
	deviceService *device.Service,
	agentProfileRepo repository.AgentProfileRepository,
	modelProfileRepo repository.ModelProfileRepository,
	policyProfileRepo repository.PolicyProfileRepository,
) *LifecycleHandler {
	return &LifecycleHandler{
		deviceService:     deviceService,
		agentProfileRepo:  agentProfileRepo,
		modelProfileRepo:  modelProfileRepo,
		policyProfileRepo: policyProfileRepo,
	}
}

func (h *LifecycleHandler) RegisterRoutes(router *gin.RouterGroup) {
	agentGroup := router.Group("/agent/v1")
	{
		agentGroup.POST("/heartbeat", h.Heartbeat)
		agentGroup.GET("/config", h.GetConfig)
	}
}

func (h *LifecycleHandler) Heartbeat(c *gin.Context) {
	var req dto.AgentHeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		mw.RespondValidationError(c, err.Error())
		return
	}

	deviceID := req.DeviceID
	if ctxDeviceID, exists := c.Get("device_id"); exists {
		deviceID = ctxDeviceID.(string)
	}

	device, err := h.deviceService.Heartbeat(c.Request.Context(), deviceID, req.AgentVersion, req.PolicyVersion)
	if err != nil {
		mw.RespondError(c, err)
		return
	}

	mw.RespondSuccess(c, http.StatusOK, gin.H{
		"status":         "ok",
		"config_version": device.PolicyVersion,
	})
}

func (h *LifecycleHandler) GetConfig(c *gin.Context) {
	deviceID := c.Query("device_id")
	if ctxDeviceID, exists := c.Get("device_id"); exists {
		deviceID = ctxDeviceID.(string)
	}
	if deviceID == "" {
		mw.RespondValidationError(c, "device_id is required")
		return
	}

	dev, err := h.deviceService.GetConfig(c.Request.Context(), deviceID)
	if err != nil {
		mw.RespondError(c, err)
		return
	}

	var agentProfile interface{}
	var modelProfile interface{}
	var policyProfile interface{}

	if dev.AgentProfileID != "" {
		ap, _ := h.agentProfileRepo.GetByID(c.Request.Context(), dev.AgentProfileID, dev.TenantID)
		if ap != nil {
			agentProfile = ap
			mp, _ := h.modelProfileRepo.GetByID(c.Request.Context(), ap.ModelProfileID, dev.TenantID)
			modelProfile = mp
			pp, _ := h.policyProfileRepo.GetByID(c.Request.Context(), ap.PolicyProfileID, dev.TenantID)
			policyProfile = pp
		}
	}

	currentVersion := 0
	if v := c.Query("current_config_version"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			currentVersion = parsed
		}
	}

	mw.RespondSuccess(c, http.StatusOK, dto.AgentConfigResponse{
		HasUpdate:     dev.PolicyVersion > currentVersion,
		ConfigVersion: dev.PolicyVersion,
		AgentProfile:  agentProfile,
		ModelProfile:  modelProfile,
		PolicyProfile: policyProfile,
	})
}

// deviceStatusFromRequest maps an agent-reported status string to a domain status.
func deviceStatusFromRequest(s string) domain.DeviceStatus {
	switch s {
	case "active":
		return domain.DeviceStatusActive
	case "quarantined":
		return domain.DeviceStatusQuarantined
	default:
		return domain.DeviceStatusActive
	}
}
