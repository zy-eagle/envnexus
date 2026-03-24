package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/dto"
	mw "github.com/zy-eagle/envnexus/services/platform-api/internal/middleware"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/tenant"
)

type TenantHandler struct {
	tenantService *tenant.Service
}

func NewTenantHandler(tenantService *tenant.Service) *TenantHandler {
	return &TenantHandler{
		tenantService: tenantService,
	}
}

func (h *TenantHandler) RegisterRoutes(router *gin.RouterGroup) {
	tenants := router.Group("/tenants")
	{
		tenants.POST("", h.CreateTenant)
		tenants.GET("", h.ListTenants)
		tenants.GET("/:tenantId", h.GetTenant)
		tenants.PUT("/:tenantId", h.UpdateTenant)
		tenants.DELETE("/:tenantId", h.DeleteTenant)
	}
}

func (h *TenantHandler) CreateTenant(c *gin.Context) {
	var req dto.CreateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		mw.RespondValidationError(c, err.Error())
		return
	}

	resp, err := h.tenantService.CreateTenant(c.Request.Context(), req)
	if err != nil {
		mw.RespondError(c, err)
		return
	}

	mw.RespondSuccess(c, http.StatusCreated, resp)
}

func (h *TenantHandler) GetTenant(c *gin.Context) {
	id := c.Param("tenantId")

	resp, err := h.tenantService.GetTenant(c.Request.Context(), id)
	if err != nil {
		mw.RespondError(c, err)
		return
	}

	mw.RespondSuccess(c, http.StatusOK, resp)
}

func (h *TenantHandler) ListTenants(c *gin.Context) {
	resp, err := h.tenantService.ListTenants(c.Request.Context())
	if err != nil {
		mw.RespondError(c, err)
		return
	}

	mw.RespondSuccess(c, http.StatusOK, resp)
}

func (h *TenantHandler) UpdateTenant(c *gin.Context) {
	id := c.Param("tenantId")

	var req dto.UpdateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		mw.RespondValidationError(c, err.Error())
		return
	}

	resp, err := h.tenantService.UpdateTenant(c.Request.Context(), id, req)
	if err != nil {
		mw.RespondError(c, err)
		return
	}

	mw.RespondSuccess(c, http.StatusOK, resp)
}

func (h *TenantHandler) DeleteTenant(c *gin.Context) {
	id := c.Param("tenantId")

	err := h.tenantService.DeleteTenant(c.Request.Context(), id)
	if err != nil {
		mw.RespondError(c, err)
		return
	}

	mw.RespondSuccess(c, http.StatusOK, gin.H{"status": "deleted"})
}
