package agent

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/zy-eagle/envnexus/services/platform-api/internal/domain"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/dto"
	mw "github.com/zy-eagle/envnexus/services/platform-api/internal/middleware"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/repository"
)

type LifecycleHandler struct {
	deviceRepo       repository.DeviceRepository
	agentProfileRepo repository.AgentProfileRepository
	modelProfileRepo repository.ModelProfileRepository
	policyProfileRepo repository.PolicyProfileRepository
}

func NewLifecycleHandler(
	deviceRepo repository.DeviceRepository,
	agentProfileRepo repository.AgentProfileRepository,
	modelProfileRepo repository.ModelProfileRepository,
	policyProfileRepo repository.PolicyProfileRepository,
) *LifecycleHandler {
	return &LifecycleHandler{
		deviceRepo:        deviceRepo,
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

	device, err := h.deviceRepo.GetByID(c.Request.Context(), deviceID)
	if err != nil || device == nil {
		mw.RespondError(c, domain.ErrDeviceNotFound)
		return
	}
	if device.IsRevoked() {
		mw.RespondError(c, domain.ErrDeviceRevoked)
		return
	}

	device.RecordHeartbeat(req.AgentVersion, req.PolicyVersion)
	if err := h.deviceRepo.Update(c.Request.Context(), device); err != nil {
		mw.RespondError(c, domain.ErrInternalError)
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

	device, err := h.deviceRepo.GetByID(c.Request.Context(), deviceID)
	if err != nil || device == nil {
		mw.RespondError(c, domain.ErrDeviceNotFound)
		return
	}

	var agentProfile interface{}
	var modelProfile interface{}
	var policyProfile interface{}

	if device.AgentProfileID != "" {
		ap, _ := h.agentProfileRepo.GetByID(c.Request.Context(), device.AgentProfileID, device.TenantID)
		if ap != nil {
			agentProfile = ap
			mp, _ := h.modelProfileRepo.GetByID(c.Request.Context(), ap.ModelProfileID, device.TenantID)
			modelProfile = mp
			pp, _ := h.policyProfileRepo.GetByID(c.Request.Context(), ap.PolicyProfileID, device.TenantID)
			policyProfile = pp
		}
	}

	currentVersion := 0
	if v := c.Query("current_config_version"); v != "" {
		// Simple parse; ignore errors for MVP
		for _, ch := range v {
			currentVersion = currentVersion*10 + int(ch-'0')
		}
	}

	hasUpdate := device.PolicyVersion > currentVersion

	mw.RespondSuccess(c, http.StatusOK, dto.AgentConfigResponse{
		HasUpdate:     hasUpdate,
		ConfigVersion: device.PolicyVersion,
		AgentProfile:  agentProfile,
		ModelProfile:  modelProfile,
		PolicyProfile: policyProfile,
	})
}
