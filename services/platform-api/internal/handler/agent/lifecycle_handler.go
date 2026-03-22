package agent

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type LifecycleHandler struct {
}

func NewLifecycleHandler() *LifecycleHandler {
	return &LifecycleHandler{}
}

func (h *LifecycleHandler) RegisterRoutes(router *gin.RouterGroup) {
	agentGroup := router.Group("/agent/v1")
	{
		agentGroup.POST("/heartbeat", h.Heartbeat)
		agentGroup.GET("/config", h.GetConfig)
	}
}

func (h *LifecycleHandler) Heartbeat(c *gin.Context) {
	// TODO: implement actual heartbeat logic (update device last_seen_at, etc.)
	// For MVP, just return success
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

func (h *LifecycleHandler) GetConfig(c *gin.Context) {
	// TODO: implement actual config retrieval based on device's tenant and agent profile
	// For MVP, return a mock config
	c.JSON(http.StatusOK, gin.H{
		"config_version": "v1.0.0",
		"agent_profile": gin.H{
			"id": "ap_mock123",
			"name": "Default Profile",
			"log_level": "info",
		},
		"policy_profile": gin.H{
			"id": "pp_mock123",
			"name": "Default Policy",
			"auto_approve_low_risk": true,
		},
	})
}
