package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/dto"
	mw "github.com/zy-eagle/envnexus/services/platform-api/internal/middleware"
	package_svc "github.com/zy-eagle/envnexus/services/platform-api/internal/service/package"
)

type PackageHandler struct {
	pkgService *package_svc.Service
}

func NewPackageHandler(pkgService *package_svc.Service) *PackageHandler {
	return &PackageHandler{
		pkgService: pkgService,
	}
}

func (h *PackageHandler) RegisterRoutes(router *gin.RouterGroup) {
	pkgs := router.Group("/tenants/:tenantId/download-packages")
	{
		pkgs.POST("", h.CreatePackage)
		pkgs.GET("", h.ListPackages)
	}
}

func (h *PackageHandler) CreatePackage(c *gin.Context) {
	tenantID := c.Param("tenantId")
	if tenantID == "" {
		mw.RespondValidationError(c, "tenantId is required")
		return
	}

	var req dto.CreatePackageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		mw.RespondValidationError(c, err.Error())
		return
	}

	resp, err := h.pkgService.CreatePackage(c.Request.Context(), tenantID, req)
	if err != nil {
		mw.RespondError(c, err)
		return
	}

	mw.RespondSuccess(c, http.StatusCreated, resp)
}

func (h *PackageHandler) ListPackages(c *gin.Context) {
	tenantID := c.Param("tenantId")
	if tenantID == "" {
		mw.RespondValidationError(c, "tenantId is required")
		return
	}

	resp, err := h.pkgService.ListPackages(c.Request.Context(), tenantID)
	if err != nil {
		mw.RespondError(c, err)
		return
	}

	mw.RespondSuccess(c, http.StatusOK, resp)
}
