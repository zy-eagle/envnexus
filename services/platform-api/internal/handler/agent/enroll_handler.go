package agent

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/dto"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/enrollment"
)

type EnrollHandler struct {
	enrollService *enrollment.Service
}

func NewEnrollHandler(enrollService *enrollment.Service) *EnrollHandler {
	return &EnrollHandler{
		enrollService: enrollService,
	}
}

func (h *EnrollHandler) RegisterRoutes(router *gin.RouterGroup) {
	agentGroup := router.Group("/agent/v1")
	{
		agentGroup.POST("/enroll", h.Enroll)
	}
}

func (h *EnrollHandler) Enroll(c *gin.Context) {
	var req dto.AgentEnrollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.enrollService.EnrollAgent(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}
