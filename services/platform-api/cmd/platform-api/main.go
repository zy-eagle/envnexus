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
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/enrollment"
	package_svc "github.com/zy-eagle/envnexus/services/platform-api/internal/service/package"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/tenant"
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

	tenantHandler := httphandler.NewTenantHandler(tenantService)
	tokenHandler := httphandler.NewTokenHandler(enrollService)
	pkgHandler := httphandler.NewPackageHandler(pkgService)
	agentEnrollHandler := agent.NewEnrollHandler(enrollService)
	agentAuditHandler := agent.NewAuditHandler(auditService)

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
	}

	// Agent API group
	agentGroup := router.Group("")
	agentEnrollHandler.RegisterRoutes(agentGroup)
	agentAuditHandler.RegisterRoutes(agentGroup)

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
