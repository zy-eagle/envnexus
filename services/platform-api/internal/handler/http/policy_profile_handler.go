package http

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/dto"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/policy_profile"
)

type PolicyProfileHandler struct {
	profileService *policy_profile.Service
}

func NewPolicyProfileHandler(profileService *policy_profile.Service) *PolicyProfileHandler {
	return &PolicyProfileHandler{
		profileService: profileService,
	}
}

func (h *PolicyProfileHandler) RegisterRoutes(router *gin.RouterGroup) {
	profiles := router.Group("/tenants/:tenantId/policy-profiles")
	{
		profiles.GET("", h.ListProfiles)
		profiles.POST("", h.CreateProfile)
		profiles.PUT("/:id", h.UpdateProfile)
		profiles.DELETE("/:id", h.DeleteProfile)
	}
}

func (h *PolicyProfileHandler) ListProfiles(c *gin.Context) {
	tenantID := c.Param("tenantId")
	resp, err := h.profileService.ListProfiles(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": resp})
}

func (h *PolicyProfileHandler) CreateProfile(c *gin.Context) {
	tenantID := c.Param("tenantId")
	var req dto.CreatePolicyProfileRequest
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

func (h *PolicyProfileHandler) UpdateProfile(c *gin.Context) {
	tenantID := c.Param("tenantId")
	id := c.Param("id")

	var req dto.UpdatePolicyProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.profileService.UpdateProfile(c.Request.Context(), tenantID, id, req)
	if err != nil {
		if err == context.Canceled {
			c.JSON(http.StatusNotFound, gin.H{"error": "profile not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": resp})
}

func (h *PolicyProfileHandler) DeleteProfile(c *gin.Context) {
	tenantID := c.Param("tenantId")
	id := c.Param("id")

	err := h.profileService.DeleteProfile(c.Request.Context(), tenantID, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}
