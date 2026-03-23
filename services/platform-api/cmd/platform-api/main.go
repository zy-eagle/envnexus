package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/zy-eagle/envnexus/services/platform-api/internal/handler/agent"
	httphandler "github.com/zy-eagle/envnexus/services/platform-api/internal/handler/http"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/middleware"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/repository"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/agent_profile"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/audit"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/auth"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/device"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/enrollment"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/model_profile"
	package_svc "github.com/zy-eagle/envnexus/services/platform-api/internal/service/package"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/policy_profile"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/session"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/tenant"
)

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	dsn := os.Getenv("ENX_DATABASE_DSN")
	jwtSecret := envOrDefault("ENX_JWT_SECRET", "dev-jwt-secret-change-me")
	deviceSecret := envOrDefault("ENX_DEVICE_TOKEN_SECRET", "dev-device-secret-change-me")
	httpPort := envOrDefault("ENX_HTTP_PORT", "8080")

	// --- Repositories ---
	var (
		tenantRepo        repository.TenantRepository
		enrollRepo        repository.EnrollmentRepository
		deviceRepo        repository.DeviceRepository
		auditRepo         repository.AuditRepository
		pkgRepo           repository.PackageRepository
		userRepo          repository.UserRepository
		modelProfileRepo  repository.ModelProfileRepository
		policyProfileRepo repository.PolicyProfileRepository
		agentProfileRepo  repository.AgentProfileRepository
		sessionRepo       repository.SessionRepository
		approvalRepo      repository.ApprovalRequestRepository
		roleRepo          repository.RoleRepository
		toolInvRepo       repository.ToolInvocationRepository
	)

	if dsn != "" {
		db, err := repository.NewDB(dsn)
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}
		tenantRepo = repository.NewMySQLTenantRepository(db)
		enrollRepo = repository.NewMySQLEnrollmentRepository(db)
		deviceRepo = repository.NewMySQLDeviceRepository(db)
		auditRepo = repository.NewMySQLAuditRepository(db)
		pkgRepo = repository.NewMySQLPackageRepository(db)
		userRepo = repository.NewMySQLUserRepository(db)
		modelProfileRepo = repository.NewMySQLModelProfileRepository(db)
		policyProfileRepo = repository.NewMySQLPolicyProfileRepository(db)
		agentProfileRepo = repository.NewMySQLAgentProfileRepository(db)
		sessionRepo = repository.NewMySQLSessionRepository(db)
		approvalRepo = repository.NewMySQLApprovalRequestRepository(db)
		roleRepo = repository.NewMySQLRoleRepository(db)
		toolInvRepo = repository.NewMySQLToolInvocationRepository(db)
		log.Println("Connected to MySQL database")
	} else {
		log.Println("ENX_DATABASE_DSN not set, using in-memory tenant repo (limited functionality)")
		tenantRepo = repository.NewMemoryTenantRepository()
	}

	// Suppress unused variable warnings for repos that are only used with DB
	_ = roleRepo
	_ = toolInvRepo

	// --- Services ---
	authService := auth.NewService(userRepo, jwtSecret, deviceSecret)
	tenantService := tenant.NewService(tenantRepo)
	enrollService := enrollment.NewService(enrollRepo, deviceRepo, authService)
	auditService := audit.NewService(auditRepo)
	pkgService := package_svc.NewService(pkgRepo)
	modelProfileService := model_profile.NewService(modelProfileRepo)
	policyProfileService := policy_profile.NewService(policyProfileRepo)
	agentProfileService := agent_profile.NewService(agentProfileRepo)
	deviceService := device.NewService(deviceRepo)
	sessionService := session.NewService(sessionRepo, approvalRepo, deviceRepo, auditRepo)

	// --- Handlers ---
	tenantHandler := httphandler.NewTenantHandler(tenantService)
	tokenHandler := httphandler.NewTokenHandler(enrollService)
	pkgHandler := httphandler.NewPackageHandler(pkgService)
	authHandler := httphandler.NewAuthHandler(authService)
	modelProfileHandler := httphandler.NewModelProfileHandler(modelProfileService)
	policyProfileHandler := httphandler.NewPolicyProfileHandler(policyProfileService)
	agentProfileHandler := httphandler.NewAgentProfileHandler(agentProfileService)
	deviceHandler := httphandler.NewDeviceHandler(deviceService)
	sessionHandler := httphandler.NewSessionHandler(sessionService)
	auditHandler := httphandler.NewAuditHandler(auditService)

	agentEnrollHandler := agent.NewEnrollHandler(enrollService)
	agentAuditHandler := agent.NewAuditHandler(auditRepo)
	agentLifecycleHandler := agent.NewLifecycleHandler(deviceRepo, agentProfileRepo, modelProfileRepo, policyProfileRepo)

	// --- Router ---
	router := gin.Default()

	router.GET("/healthz", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})
	router.GET("/readyz", func(c *gin.Context) {
		if dsn == "" {
			c.String(http.StatusServiceUnavailable, "No database configured")
			return
		}
		c.String(http.StatusOK, "Ready")
	})

	// Public endpoints (no auth required)
	publicV1 := router.Group("/api/v1")
	{
		publicV1.POST("/auth/login", func(c *gin.Context) { authHandler.Login(c) })
		publicV1.GET("/bootstrap", func(c *gin.Context) { authHandler.Bootstrap(c) })
	}

	// Protected console API
	protectedV1 := router.Group("/api/v1")
	protectedV1.Use(middleware.JWTAuth(jwtSecret))
	{
		protectedV1.GET("/me", func(c *gin.Context) { authHandler.Me(c) })
		tenantHandler.RegisterRoutes(protectedV1)
		tokenHandler.RegisterRoutes(protectedV1)
		pkgHandler.RegisterRoutes(protectedV1)
		modelProfileHandler.RegisterRoutes(protectedV1)
		policyProfileHandler.RegisterRoutes(protectedV1)
		agentProfileHandler.RegisterRoutes(protectedV1)
		deviceHandler.RegisterRoutes(protectedV1)
		sessionHandler.RegisterRoutes(protectedV1)
		auditHandler.RegisterRoutes(protectedV1)
	}

	// Agent API (device token or open for enrollment)
	agentEnrollHandler.RegisterRoutes(router.Group(""))
	agentDeviceGroup := router.Group("")
	agentDeviceGroup.Use(middleware.DeviceAuth(deviceSecret))
	agentAuditHandler.RegisterRoutes(agentDeviceGroup)
	agentLifecycleHandler.RegisterRoutes(agentDeviceGroup)

	// --- Server ---
	server := &http.Server{
		Addr:         ":" + httpPort,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		log.Printf("Starting platform-api on :%s", httpPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Server exited")
}
