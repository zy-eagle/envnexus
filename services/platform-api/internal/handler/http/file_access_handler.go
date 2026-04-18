package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/zy-eagle/envnexus/services/platform-api/internal/domain"
	mw "github.com/zy-eagle/envnexus/services/platform-api/internal/middleware"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/file_access"
)

type FileAccessHandler struct {
	svc *file_access.Service
}

func NewFileAccessHandler(svc *file_access.Service) *FileAccessHandler {
	return &FileAccessHandler{svc: svc}
}

func (h *FileAccessHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/tenants/:tenantId/file-access-requests", h.Create)
	router.GET("/tenants/:tenantId/file-access-requests", h.List)
	router.GET("/tenants/:tenantId/file-access-requests/:requestId", h.Get)
	router.POST("/tenants/:tenantId/file-access-requests/:requestId/approve", h.Approve)
	router.POST("/tenants/:tenantId/file-access-requests/:requestId/deny", h.Deny)
}

type createFileAccessReq struct {
	DeviceID string `json:"device_id" binding:"required"`
	Path     string `json:"path" binding:"required"`
	Action   string `json:"action" binding:"required,oneof=browse preview download"`
	Note     string `json:"note"`
}

func (h *FileAccessHandler) Create(c *gin.Context) {
	tenantID := c.Param("tenantId")
	var req createFileAccessReq
	if err := c.ShouldBindJSON(&req); err != nil {
		mw.RespondValidationError(c, err.Error())
		return
	}

	userID, _ := c.Get("user_id")
	uid, _ := userID.(string)
	if uid == "" {
		mw.RespondError(c, domain.ErrUnauthorized)
		return
	}

	far, err := h.svc.CreateRequest(c.Request.Context(), tenantID, req.DeviceID, uid, req.Path, domain.FileAccessAction(req.Action), req.Note)
	if err != nil {
		mw.RespondError(c, err)
		return
	}
	mw.RespondSuccess(c, http.StatusCreated, far)
}

func (h *FileAccessHandler) List(c *gin.Context) {
	tenantID := c.Param("tenantId")
	status := c.Query("status")
	items, err := h.svc.ListByTenant(c.Request.Context(), tenantID, status)
	if err != nil {
		mw.RespondError(c, err)
		return
	}
	mw.RespondSuccess(c, http.StatusOK, gin.H{"items": items})
}

func (h *FileAccessHandler) Get(c *gin.Context) {
	requestID := c.Param("requestId")
	far, err := h.svc.GetByID(c.Request.Context(), requestID)
	if err != nil {
		mw.RespondError(c, err)
		return
	}
	mw.RespondSuccess(c, http.StatusOK, far)
}

func (h *FileAccessHandler) Approve(c *gin.Context) {
	requestID := c.Param("requestId")
	userID, _ := c.Get("user_id")
	uid, _ := userID.(string)
	far, err := h.svc.Approve(c.Request.Context(), requestID, uid)
	if err != nil {
		mw.RespondError(c, err)
		return
	}
	mw.RespondSuccess(c, http.StatusOK, far)
}

func (h *FileAccessHandler) Deny(c *gin.Context) {
	requestID := c.Param("requestId")
	userID, _ := c.Get("user_id")
	uid, _ := userID.(string)
	far, err := h.svc.Deny(c.Request.Context(), requestID, uid)
	if err != nil {
		mw.RespondError(c, err)
		return
	}
	mw.RespondSuccess(c, http.StatusOK, far)
}
