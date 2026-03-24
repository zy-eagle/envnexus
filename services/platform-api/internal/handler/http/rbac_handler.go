package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/middleware"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/rbac"
)

type RBACHandler struct {
	svc *rbac.Service
}

func NewRBACHandler(svc *rbac.Service) *RBACHandler {
	return &RBACHandler{svc: svc}
}

func (h *RBACHandler) RegisterRoutes(rg *gin.RouterGroup) {
	tenants := rg.Group("/tenants/:tenantId")

	// Roles
	tenants.GET("/roles", h.ListRoles)
	tenants.POST("/roles", h.CreateRole)
	tenants.PUT("/roles/:roleId", h.UpdateRole)
	tenants.DELETE("/roles/:roleId", h.DeleteRole)

	// Role Bindings
	tenants.GET("/role-bindings", h.ListBindings)
	tenants.POST("/role-bindings", h.BindRole)
	tenants.DELETE("/role-bindings/:bindingId", h.UnbindRole)

	// My permissions
	rg.GET("/me/permissions", h.MyPermissions)
}

func (h *RBACHandler) ListRoles(c *gin.Context) {
	tenantID := c.Param("tenantId")
	roles, err := h.svc.ListRoles(c.Request.Context(), tenantID)
	if err != nil {
		middleware.RespondErrorCode(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	middleware.RespondSuccess(c, http.StatusOK, gin.H{"items": roles, "total": len(roles)})
}

func (h *RBACHandler) CreateRole(c *gin.Context) {
	tenantID := c.Param("tenantId")
	var req struct {
		Name        string   `json:"name" binding:"required"`
		Permissions []string `json:"permissions"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondErrorCode(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	role, err := h.svc.CreateRole(c.Request.Context(), tenantID, req.Name, req.Permissions)
	if err != nil {
		middleware.RespondErrorCode(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	middleware.RespondSuccess(c, http.StatusOK, role)
}

func (h *RBACHandler) UpdateRole(c *gin.Context) {
	roleID := c.Param("roleId")
	var req struct {
		Permissions []string `json:"permissions" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondErrorCode(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	role, err := h.svc.UpdateRole(c.Request.Context(), roleID, req.Permissions)
	if err != nil {
		middleware.RespondErrorCode(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	middleware.RespondSuccess(c, http.StatusOK, role)
}

func (h *RBACHandler) DeleteRole(c *gin.Context) {
	roleID := c.Param("roleId")
	if err := h.svc.DeleteRole(c.Request.Context(), roleID); err != nil {
		middleware.RespondErrorCode(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	middleware.RespondSuccess(c, http.StatusOK, gin.H{"deleted": roleID})
}

func (h *RBACHandler) ListBindings(c *gin.Context) {
	tenantID := c.Param("tenantId")
	bindings, err := h.svc.ListBindings(c.Request.Context(), tenantID)
	if err != nil {
		middleware.RespondErrorCode(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	middleware.RespondSuccess(c, http.StatusOK, gin.H{"items": bindings, "total": len(bindings)})
}

func (h *RBACHandler) BindRole(c *gin.Context) {
	tenantID := c.Param("tenantId")
	callerID, _ := c.Get("user_id")
	var req struct {
		UserID string `json:"user_id" binding:"required"`
		RoleID string `json:"role_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondErrorCode(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	rb, err := h.svc.BindRole(c.Request.Context(), tenantID, req.UserID, req.RoleID, callerID.(string))
	if err != nil {
		middleware.RespondErrorCode(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	middleware.RespondSuccess(c, http.StatusOK, rb)
}

func (h *RBACHandler) UnbindRole(c *gin.Context) {
	var req struct {
		UserID string `json:"user_id" binding:"required"`
		RoleID string `json:"role_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondErrorCode(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if err := h.svc.UnbindRole(c.Request.Context(), req.UserID, req.RoleID); err != nil {
		middleware.RespondErrorCode(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	middleware.RespondSuccess(c, http.StatusOK, gin.H{"ok": true})
}

func (h *RBACHandler) MyPermissions(c *gin.Context) {
	userID, _ := c.Get("user_id")
	perms, err := h.svc.GetUserPermissions(c.Request.Context(), userID.(string))
	if err != nil {
		middleware.RespondErrorCode(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	middleware.RespondSuccess(c, http.StatusOK, gin.H{"permissions": perms})
}

