package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/dto"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/model_profile"
)

type ModelProfileHandler struct {
	profileService *model_profile.Service
}

func NewModelProfileHandler(profileService *model_profile.Service) *ModelProfileHandler {
	return &ModelProfileHandler{
		profileService: profileService,
	}
}

func (h *ModelProfileHandler) RegisterRoutes(router *gin.RouterGroup) {
	profiles := router.Group("/tenants/:tenantId/model-profiles")
	{
		profiles.GET("", h.ListProfiles)
		profiles.POST("", h.CreateProfile)
	}
}

func (h *ModelProfileHandler) ListProfiles(c *gin.Context) {
	tenantID := c.Param("tenantId")
	resp, err := h.profileService.ListProfiles(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": resp})
}

func (h *ModelProfileHandler) CreateProfile(c *gin.Context) {
	tenantID := c.Param("tenantId")
	var req dto.CreateModelProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.profileService.CreateProfile(c.Request.Context(), tenantID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": resp})
}
