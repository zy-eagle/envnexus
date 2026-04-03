package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/dto"
	mw "github.com/zy-eagle/envnexus/services/platform-api/internal/middleware"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/user"
)

type UserHandler struct {
	svc *user.Service
}

func NewUserHandler(svc *user.Service) *UserHandler {
	return &UserHandler{svc: svc}
}

func (h *UserHandler) RegisterRoutes(rg *gin.RouterGroup) {
	tenants := rg.Group("/tenants/:tenantId")
	{
		tenants.GET("/users", h.ListUsers)
		tenants.POST("/users", h.CreateUser)
		tenants.PUT("/users/:userId", h.UpdateUser)
		tenants.DELETE("/users/:userId", h.DeleteUser)
	}
}

func (h *UserHandler) ListUsers(c *gin.Context) {
	tenantID := c.Param("tenantId")
	q := c.Query("q")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	items, err := h.svc.List(c.Request.Context(), tenantID, q, limit)
	if err != nil {
		mw.RespondError(c, err)
		return
	}
	mw.RespondSuccess(c, http.StatusOK, gin.H{"items": items, "total": len(items)})
}

func (h *UserHandler) CreateUser(c *gin.Context) {
	tenantID := c.Param("tenantId")
	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		mw.RespondValidationError(c, err.Error())
		return
	}
	resp, err := h.svc.Create(c.Request.Context(), tenantID, req)
	if err != nil {
		mw.RespondError(c, err)
		return
	}
	mw.RespondSuccess(c, http.StatusCreated, resp)
}

func (h *UserHandler) UpdateUser(c *gin.Context) {
	tenantID := c.Param("tenantId")
	userID := c.Param("userId")
	var req dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		mw.RespondValidationError(c, err.Error())
		return
	}
	resp, err := h.svc.Update(c.Request.Context(), tenantID, userID, req)
	if err != nil {
		mw.RespondError(c, err)
		return
	}
	mw.RespondSuccess(c, http.StatusOK, resp)
}

func (h *UserHandler) DeleteUser(c *gin.Context) {
	tenantID := c.Param("tenantId")
	userID := c.Param("userId")
	if err := h.svc.Delete(c.Request.Context(), tenantID, userID); err != nil {
		mw.RespondError(c, err)
		return
	}
	mw.RespondSuccess(c, http.StatusOK, gin.H{"deleted": userID})
}

