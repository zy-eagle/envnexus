package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/zy-eagle/envnexus/services/platform-api/internal/domain"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/dto"
	mw "github.com/zy-eagle/envnexus/services/platform-api/internal/middleware"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/marketplace"
)

type MarketplaceHandler struct {
	svc *marketplace.Service
}

func NewMarketplaceHandler(svc *marketplace.Service) *MarketplaceHandler {
	return &MarketplaceHandler{svc: svc}
}

func (h *MarketplaceHandler) RegisterRoutes(router *gin.RouterGroup) {
	m := router.Group("/tenants/:tenantId/marketplace")
	{
		m.GET("/items", h.ListItems)
		m.GET("/subscriptions", h.ListSubscriptions)
		m.POST("/subscriptions", h.Subscribe)
		m.DELETE("/subscriptions/:itemId", h.Unsubscribe)
	}
}

func (h *MarketplaceHandler) requireTenantScope(c *gin.Context, tenantID string) bool {
	jwtTenant, ok := c.Get("tenant_id")
	var super bool
	if v, ok2 := c.Get("platform_super_admin"); ok2 {
		if b, ok3 := v.(bool); ok3 {
			super = b
		}
	}
	if !ok {
		mw.RespondErrorCode(c, http.StatusUnauthorized, "unauthorized", "missing tenant context")
		return false
	}
	jt, ok := jwtTenant.(string)
	if !ok {
		mw.RespondErrorCode(c, http.StatusUnauthorized, "unauthorized", "invalid tenant context")
		return false
	}
	if jt != tenantID && !super {
		mw.RespondErrorCode(c, http.StatusForbidden, "forbidden", "tenant scope mismatch")
		return false
	}
	return true
}

func (h *MarketplaceHandler) ListItems(c *gin.Context) {
	tenantID := c.Param("tenantId")
	if !h.requireTenantScope(c, tenantID) {
		return
	}
	var itemType *domain.MarketplaceItemType
	if t := c.Query("type"); t != "" {
		v := domain.MarketplaceItemType(t)
		itemType = &v
	}
	var status *domain.MarketplaceItemStatus
	if s := c.Query("status"); s != "" {
		v := domain.MarketplaceItemStatus(s)
		status = &v
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	items, total, err := h.svc.ListItems(c.Request.Context(), itemType, status, page, pageSize)
	if err != nil {
		mw.RespondError(c, err)
		return
	}
	mw.RespondSuccess(c, http.StatusOK, gin.H{"items": items, "total": total, "page": page, "page_size": pageSize})
}

func (h *MarketplaceHandler) ListSubscriptions(c *gin.Context) {
	tenantID := c.Param("tenantId")
	if !h.requireTenantScope(c, tenantID) {
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	subs, total, err := h.svc.ListSubscriptions(c.Request.Context(), tenantID, page, pageSize)
	if err != nil {
		mw.RespondError(c, err)
		return
	}
	mw.RespondSuccess(c, http.StatusOK, gin.H{"items": subs, "total": total, "page": page, "page_size": pageSize})
}

func (h *MarketplaceHandler) Subscribe(c *gin.Context) {
	tenantID := c.Param("tenantId")
	if !h.requireTenantScope(c, tenantID) {
		return
	}
	var req dto.MarketplaceSubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		mw.RespondValidationError(c, err.Error())
		return
	}
	out, err := h.svc.Subscribe(c.Request.Context(), tenantID, req.ItemID)
	if err != nil {
		mw.RespondError(c, err)
		return
	}
	mw.RespondSuccess(c, http.StatusCreated, out)
}

func (h *MarketplaceHandler) Unsubscribe(c *gin.Context) {
	tenantID := c.Param("tenantId")
	if !h.requireTenantScope(c, tenantID) {
		return
	}
	itemID := c.Param("itemId")
	if itemID == "" {
		mw.RespondValidationError(c, "itemId is required")
		return
	}
	if err := h.svc.Unsubscribe(c.Request.Context(), tenantID, itemID); err != nil {
		mw.RespondError(c, err)
		return
	}
	mw.RespondSuccess(c, http.StatusOK, gin.H{"status": "ok"})
}
