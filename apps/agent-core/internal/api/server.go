package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zy-eagle/envnexus/apps/agent-core/internal/device"
	"github.com/zy-eagle/envnexus/apps/agent-core/internal/policy"
)

type LocalServer struct {
	port            int
	server          *http.Server
	identityManager *device.IdentityManager
	policyEngine    *policy.Engine
}

func NewLocalServer(port int, identityManager *device.IdentityManager, policyEngine *policy.Engine) *LocalServer {
	return &LocalServer{
		port:            port,
		identityManager: identityManager,
		policyEngine:    policyEngine,
	}
}

func (s *LocalServer) Start() error {
	// Use release mode for local server to reduce log noise
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	// Local API routes
	api := router.Group("/local/v1")
	{
		api.GET("/runtime/status", s.handleRuntimeStatus)
		api.GET("/device", s.handleDevice)
		
		// Approval routes for local UI
		api.GET("/approvals/pending", s.handleGetPendingApprovals)
		api.POST("/approvals/:id/resolve", s.handleResolveApproval)
	}

	s.server = &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", s.port),
		Handler: router,
	}

	log.Printf("Starting local API server on %s\n", s.server.Addr)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Local API server failed: %v", err)
		}
	}()

	return nil
}

func (s *LocalServer) Stop(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

func (s *LocalServer) handleRuntimeStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "running",
		"uptime": time.Now().Unix(), // In a real app, track start time
	})
}

func (s *LocalServer) handleDevice(c *gin.Context) {
	deviceID, err := s.identityManager.GetOrCreateDeviceID()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get device identity"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"device_id": deviceID,
		// In a real app, return full device state, tenant info, etc.
	})
}

func (s *LocalServer) handleGetPendingApprovals(c *gin.Context) {
	pending := s.policyEngine.GetPending()
	c.JSON(http.StatusOK, gin.H{
		"pending_approvals": pending,
	})
}

type ResolveApprovalRequest struct {
	Approved bool `json:"approved"`
}

func (s *LocalServer) handleResolveApproval(c *gin.Context) {
	id := c.Param("id")
	
	var req ResolveApprovalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.policyEngine.Resolve(id, req.Approved); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "resolved"})
}
