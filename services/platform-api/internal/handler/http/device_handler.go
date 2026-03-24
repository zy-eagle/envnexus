package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/dto"
	mw "github.com/zy-eagle/envnexus/services/platform-api/internal/middleware"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/device"
)

func (h *DeviceHandler) RotateToken(c *gin.Context) {
	tenantID := c.Param("tenantId")
	id := c.Param("id")

	newToken, err := h.deviceService.RotateDeviceToken(c.Request.Context(), tenantID, id)
	if err != nil {
		mw.RespondError(c, err)
		return
	}
	mw.RespondSuccess(c, http.StatusOK, gin.H{"device_token": newToken})
}

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
		tenants.POST("/:id/rotate-token", h.RotateToken)
	}
}

func (h *DeviceHandler) UpdateDevice(c *gin.Context) {
	tenantID := c.Param("tenantId")
	id := c.Param("id")

	var req dto.UpdateDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		mw.RespondValidationError(c, err.Error())
		return
	}

	resp, err := h.deviceService.UpdateDevice(c.Request.Context(), tenantID, id, req)
	if err != nil {
		mw.RespondError(c, err)
		return
	}

	mw.RespondSuccess(c, http.StatusOK, resp)
}

func (h *DeviceHandler) DeleteDevice(c *gin.Context) {
	tenantID := c.Param("tenantId")
	id := c.Param("id")

	err := h.deviceService.DeleteDevice(c.Request.Context(), tenantID, id)
	if err != nil {
		mw.RespondError(c, err)
		return
	}

	mw.RespondSuccess(c, http.StatusOK, gin.H{"status": "deleted"})
}

func (h *DeviceHandler) ListDevices(c *gin.Context) {
	tenantID := c.Param("tenantId")

	resp, err := h.deviceService.ListDevices(c.Request.Context(), tenantID)
	if err != nil {
		mw.RespondError(c, err)
		return
	}

	mw.RespondSuccess(c, http.StatusOK, resp)
}
