package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zy-eagle/envnexus/apps/agent-core/internal/device"
	"github.com/zy-eagle/envnexus/apps/agent-core/internal/diagnosis"
	"github.com/zy-eagle/envnexus/apps/agent-core/internal/policy"
)

type LocalServer struct {
	port            int
	server          *http.Server
	identityManager *device.IdentityManager
	policyEngine    *policy.Engine
	diagEngine      *diagnosis.Engine
	startTime       time.Time
}

func NewLocalServer(port int, identityManager *device.IdentityManager, policyEngine *policy.Engine, diagEngine *diagnosis.Engine) *LocalServer {
	return &LocalServer{
		port:            port,
		identityManager: identityManager,
		policyEngine:    policyEngine,
		diagEngine:      diagEngine,
		startTime:       time.Now(),
	}
}

func (s *LocalServer) Start() error {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	api := router.Group("/local/v1")
	{
		api.GET("/runtime/status", s.handleRuntimeStatus)
		api.GET("/device", s.handleDevice)
		api.GET("/approvals/pending", s.handleGetPendingApprovals)
		api.POST("/approvals/:id/resolve", s.handleResolveApproval)
		api.POST("/diagnose", s.handleDiagnose)
		api.POST("/diagnostics/export", s.handleDiagnosticsExport)
	}

	s.server = &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", s.port),
		Handler: router,
	}

	slog.Info("Starting local API server", "addr", s.server.Addr)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Local API server failed", "error", err)
			os.Exit(1)
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
		"status":    "running",
		"uptime_ms": time.Since(s.startTime).Milliseconds(),
		"started":   s.startTime.Format(time.RFC3339),
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

type DiagnoseRequest struct {
	SessionID string `json:"session_id"`
	Intent    string `json:"intent" binding:"required"`
}

func (s *LocalServer) handleDiagnose(c *gin.Context) {
	var req DiagnoseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.SessionID == "" {
		req.SessionID = "local"
	}

	result, err := s.diagEngine.RunDiagnosis(c.Request.Context(), req.SessionID, req.Intent)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"diagnosis": result})
}

func (s *LocalServer) handleDiagnosticsExport(c *gin.Context) {
	report := gin.H{
		"agent_version": "0.1.0",
		"export_time":   time.Now().UTC().Format(time.RFC3339),
		"uptime_ms":     time.Since(s.startTime).Milliseconds(),
		"runtime_status": gin.H{
			"status":  "running",
			"started": s.startTime.Format(time.RFC3339),
		},
	}

	deviceID, err := s.identityManager.GetOrCreateDeviceID()
	if err == nil {
		report["device_id"] = deviceID
	}

	report["pending_approvals"] = s.policyEngine.GetPending()

	c.JSON(http.StatusOK, gin.H{"diagnostic_bundle": report})
}
