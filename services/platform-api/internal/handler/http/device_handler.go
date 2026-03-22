package http

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/dto"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/device"
)

type DeviceHandler struct {
	deviceService *device.Service
}

func NewDeviceHandler(deviceService *device.Service) *DeviceHandler {
	return &DeviceHandler{
		deviceService: deviceService,
	}
}

func (h *DeviceHandler) RegisterRoutes(router *gin.RouterGroup) {
	tenants := router.Group("/tenants/:tenantId/devices")
	{
		tenants.GET("", h.ListDevices)
		tenants.PUT("/:id", h.UpdateDevice)
		tenants.DELETE("/:id", h.DeleteDevice)
	}
}

func (h *DeviceHandler) UpdateDevice(c *gin.Context) {
	tenantID := c.Param("tenantId")
	id := c.Param("id")

	var req dto.UpdateDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.deviceService.UpdateDevice(c.Request.Context(), tenantID, id, req)
	if err != nil {
		if err == context.Canceled {
			c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": resp})
}

func (h *DeviceHandler) DeleteDevice(c *gin.Context) {
	tenantID := c.Param("tenantId")
	id := c.Param("id")

	err := h.deviceService.DeleteDevice(c.Request.Context(), tenantID, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (h *DeviceHandler) ListDevices(c *gin.Context) {
	tenantID := c.Param("tenantId")

	resp, err := h.deviceService.ListDevices(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": resp})
}
