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
	"github.com/zy-eagle/envnexus/services/platform-api/internal/repository"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/audit"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/device"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/enrollment"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/policy_profile"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/session"
	package_svc "github.com/zy-eagle/envnexus/services/platform-api/internal/service/package"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/tenant"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/agent_profile"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/auth"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/model_profile"
)

func main() {
	// 1. Load config and env vars
	// 2. Init logger, DB, Redis, Object Storage
	// 3. Run migrations
	
	// 4. Init repository
	var tenantRepo repository.TenantRepository
	var enrollRepo repository.EnrollmentRepository
	var deviceRepo repository.DeviceRepository
	var auditRepo repository.AuditRepository
	var pkgRepo repository.PackageRepository
	var userRepo repository.UserRepository
	var modelProfileRepo repository.ModelProfileRepository
	var policyProfileRepo repository.PolicyProfileRepository
	var agentProfileRepo repository.AgentProfileRepository
	var sessionRepo repository.SessionRepository

	dsn := os.Getenv("ENX_DATABASE_DSN")
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
	} else {
		log.Println("ENX_DATABASE_DSN not set, falling back to MemoryTenantRepository")
		tenantRepo = repository.NewMemoryTenantRepository()
		// For MVP, we'll panic here if no DB is provided since we didn't write memory repos for these
		log.Println("Warning: Enrollment, Device, Audit, and Package repositories require MySQL for full functionality")
	}

	// Init service, handler, middleware
	tenantService := tenant.NewService(tenantRepo)
	enrollService := enrollment.NewService(enrollRepo, deviceRepo)
	auditService := audit.NewService(auditRepo)
	pkgService := package_svc.NewService(pkgRepo)
	authService := auth.NewService(userRepo)
	modelProfileService := model_profile.NewService(modelProfileRepo)
	policyProfileService := policy_profile.NewService(policyProfileRepo)
	agentProfileService := agent_profile.NewService(agentProfileRepo)
	deviceService := device.NewService(deviceRepo)
	sessionService := session.NewService(sessionRepo)

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
	agentAuditHandler := agent.NewAuditHandler(auditService)
	agentLifecycleHandler := agent.NewLifecycleHandler()

	// 5. Register HTTP routes
	router := gin.Default()
	
	// Health checks
	router.GET("/healthz", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})
	router.GET("/readyz", func(c *gin.Context) {
		c.String(http.StatusOK, "Ready")
	})

	// API v1 group
	v1 := router.Group("/api/v1")
	{
		tenantHandler.RegisterRoutes(v1)
		tokenHandler.RegisterRoutes(v1)
		pkgHandler.RegisterRoutes(v1)
		authHandler.RegisterRoutes(v1)
		modelProfileHandler.RegisterRoutes(v1)
		policyProfileHandler.RegisterRoutes(v1)
		agentProfileHandler.RegisterRoutes(v1)
		deviceHandler.RegisterRoutes(v1)
		sessionHandler.RegisterRoutes(v1)
		auditHandler.RegisterRoutes(v1)
	}

	// Agent API group
	agentGroup := router.Group("")
	agentEnrollHandler.RegisterRoutes(agentGroup)
	agentAuditHandler.RegisterRoutes(agentGroup)
	agentLifecycleHandler.RegisterRoutes(agentGroup)

	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	// 6. Start HTTP server
	go func() {
		log.Println("Starting platform-api server on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// 7. Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}
