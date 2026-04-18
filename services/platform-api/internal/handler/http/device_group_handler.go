package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/zy-eagle/envnexus/services/platform-api/internal/domain"
	mw "github.com/zy-eagle/envnexus/services/platform-api/internal/middleware"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/device_group"
)

type DeviceGroupHandler struct {
	svc *device_group.Service
}

func NewDeviceGroupHandler(svc *device_group.Service) *DeviceGroupHandler {
	return &DeviceGroupHandler{svc: svc}
}

func (h *DeviceGroupHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/tenants/:tenantId/device-groups", h.Create)
	router.GET("/tenants/:tenantId/device-groups", h.List)
	router.GET("/tenants/:tenantId/device-groups/:groupId", h.Get)
	router.PUT("/tenants/:tenantId/device-groups/:groupId", h.Update)
	router.DELETE("/tenants/:tenantId/device-groups/:groupId", h.Delete)
	router.POST("/tenants/:tenantId/device-groups/:groupId/members", h.AddMembers)
	router.DELETE("/tenants/:tenantId/device-groups/:groupId/members/:deviceId", h.RemoveMember)
	router.GET("/tenants/:tenantId/device-groups/:groupId/members", h.ListMembers)

	router.POST("/tenants/:tenantId/batch-tasks", h.CreateBatchTask)
	router.GET("/tenants/:tenantId/batch-tasks", h.ListBatchTasks)
	router.GET("/tenants/:tenantId/batch-tasks/:taskId", h.GetBatchTask)
}

type createGroupReq struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Filter      string `json:"filter"`
}

func (h *DeviceGroupHandler) Create(c *gin.Context) {
	tenantID := c.Param("tenantId")
	var req createGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		mw.RespondValidationError(c, err.Error())
		return
	}
	userID, _ := c.Get("user_id")
	uid, _ := userID.(string)
	g, err := h.svc.CreateGroup(c.Request.Context(), tenantID, req.Name, req.Description, req.Filter, uid)
	if err != nil {
		mw.RespondError(c, err)
		return
	}
	mw.RespondSuccess(c, http.StatusCreated, g)
}

func (h *DeviceGroupHandler) List(c *gin.Context) {
	tenantID := c.Param("tenantId")
	groups, err := h.svc.ListGroups(c.Request.Context(), tenantID)
	if err != nil {
		mw.RespondError(c, err)
		return
	}
	mw.RespondSuccess(c, http.StatusOK, gin.H{"items": groups})
}

func (h *DeviceGroupHandler) Get(c *gin.Context) {
	groupID := c.Param("groupId")
	g, err := h.svc.GetGroup(c.Request.Context(), groupID)
	if err != nil {
		mw.RespondError(c, err)
		return
	}
	mw.RespondSuccess(c, http.StatusOK, g)
}

type updateGroupReq struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Filter      string `json:"filter"`
}

func (h *DeviceGroupHandler) Update(c *gin.Context) {
	groupID := c.Param("groupId")
	var req updateGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		mw.RespondValidationError(c, err.Error())
		return
	}
	g, err := h.svc.UpdateGroup(c.Request.Context(), groupID, req.Name, req.Description, req.Filter)
	if err != nil {
		mw.RespondError(c, err)
		return
	}
	mw.RespondSuccess(c, http.StatusOK, g)
}

func (h *DeviceGroupHandler) Delete(c *gin.Context) {
	groupID := c.Param("groupId")
	if err := h.svc.DeleteGroup(c.Request.Context(), groupID); err != nil {
		mw.RespondError(c, err)
		return
	}
	mw.RespondSuccess(c, http.StatusOK, gin.H{"deleted": true})
}

type addMembersReq struct {
	DeviceIDs []string `json:"device_ids" binding:"required,min=1"`
}

func (h *DeviceGroupHandler) AddMembers(c *gin.Context) {
	groupID := c.Param("groupId")
	var req addMembersReq
	if err := c.ShouldBindJSON(&req); err != nil {
		mw.RespondValidationError(c, err.Error())
		return
	}
	if err := h.svc.AddMembers(c.Request.Context(), groupID, req.DeviceIDs); err != nil {
		mw.RespondError(c, err)
		return
	}
	mw.RespondSuccess(c, http.StatusOK, gin.H{"added": len(req.DeviceIDs)})
}

func (h *DeviceGroupHandler) RemoveMember(c *gin.Context) {
	groupID := c.Param("groupId")
	deviceID := c.Param("deviceId")
	if err := h.svc.RemoveMember(c.Request.Context(), groupID, deviceID); err != nil {
		mw.RespondError(c, err)
		return
	}
	mw.RespondSuccess(c, http.StatusOK, gin.H{"removed": true})
}

func (h *DeviceGroupHandler) ListMembers(c *gin.Context) {
	groupID := c.Param("groupId")
	members, err := h.svc.ListMembers(c.Request.Context(), groupID)
	if err != nil {
		mw.RespondError(c, err)
		return
	}
	mw.RespondSuccess(c, http.StatusOK, gin.H{"items": members})
}

type createBatchTaskReq struct {
	DeviceGroupID string `json:"device_group_id" binding:"required"`
	CommandTaskID string `json:"command_task_id" binding:"required"`
	Strategy      string `json:"strategy"`
	BatchSize     int    `json:"batch_size"`
}

func (h *DeviceGroupHandler) CreateBatchTask(c *gin.Context) {
	tenantID := c.Param("tenantId")
	var req createBatchTaskReq
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
	bt, err := h.svc.CreateBatchTask(c.Request.Context(), tenantID, req.DeviceGroupID, req.CommandTaskID, req.Strategy, uid, req.BatchSize)
	if err != nil {
		mw.RespondError(c, err)
		return
	}
	mw.RespondSuccess(c, http.StatusCreated, bt)
}

func (h *DeviceGroupHandler) ListBatchTasks(c *gin.Context) {
	tenantID := c.Param("tenantId")
	tasks, err := h.svc.ListBatchTasks(c.Request.Context(), tenantID)
	if err != nil {
		mw.RespondError(c, err)
		return
	}
	mw.RespondSuccess(c, http.StatusOK, gin.H{"items": tasks})
}

func (h *DeviceGroupHandler) GetBatchTask(c *gin.Context) {
	taskID := c.Param("taskId")
	bt, err := h.svc.GetBatchTask(c.Request.Context(), taskID)
	if err != nil {
		mw.RespondError(c, err)
		return
	}
	mw.RespondSuccess(c, http.StatusOK, bt)
}
