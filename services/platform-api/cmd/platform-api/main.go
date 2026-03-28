package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/zy-eagle/envnexus/services/platform-api/internal/handler/agent"
	httphandler "github.com/zy-eagle/envnexus/services/platform-api/internal/handler/http"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/infrastructure"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/integration/feishu"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/middleware"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/repository"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/agent_profile"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/audit"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/auth"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/device"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/enrollment"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/license"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/metrics"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/model_profile"
	device_binding "github.com/zy-eagle/envnexus/services/platform-api/internal/service/device_binding"
	package_svc "github.com/zy-eagle/envnexus/services/platform-api/internal/service/package"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/policy_profile"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/rbac"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/session"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/tenant"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/webhook"
	"github.com/zy-eagle/envnexus/services/platform-api/migrations"
)

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envRequired(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("FATAL: required environment variable %s is not set", key)
	}
	return v
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	dsn := os.Getenv("ENX_DATABASE_DSN")
	jwtSecret := envRequired("ENX_JWT_SECRET")
	deviceSecret := envRequired("ENX_DEVICE_TOKEN_SECRET")
	sessionSecret := envRequired("ENX_SESSION_TOKEN_SECRET")
	httpPort := envOrDefault("ENX_HTTP_PORT", "8080")
	corsOrigins := strings.Split(envOrDefault("ENX_CORS_ALLOWED_ORIGINS", "http://localhost:3000"), ",")
	redisAddr := envOrDefault("ENX_REDIS_ADDR", "localhost:6379")
	redisPassword := os.Getenv("ENX_REDIS_PASSWORD")
	minioEndpoint := os.Getenv("ENX_OBJECT_STORAGE_ENDPOINT")
	minioPublicEndpoint := os.Getenv("ENX_OBJECT_STORAGE_PUBLIC_ENDPOINT")
	minioAccessKey := envOrDefault("MINIO_ROOT_USER", "minioadmin")
	minioSecretKey := envOrDefault("MINIO_ROOT_PASSWORD", "minioadmin")
	minioBucket := envOrDefault("ENX_OBJECT_STORAGE_BUCKET", "envnexus")
	gatewayURL := envOrDefault("ENX_GATEWAY_URL", "http://localhost:8081")
	feishuAppID := os.Getenv("ENX_FEISHU_APP_ID")
	feishuAppSecret := os.Getenv("ENX_FEISHU_APP_SECRET")
	feishuVerifyToken := os.Getenv("ENX_FEISHU_VERIFICATION_TOKEN")

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
		rbindingRepo      repository.RoleBindingRepository
		webhookSubRepo    repository.WebhookSubscriptionRepository
		webhookDelRepo    repository.WebhookDeliveryRepository
		toolInvRepo       repository.ToolInvocationRepository
		bindingRepo       repository.DeviceBindingRepository
	)

	var gormDB *gorm.DB
	if dsn != "" {
		var err error
		gormDB, err = repository.NewDB(dsn)
		if err != nil {
			slog.Error("failed to connect to database", "error", err)
			os.Exit(1)
		}
		db := gormDB
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
		rbindingRepo = repository.NewMySQLRoleBindingRepository(db)
		webhookSubRepo = repository.NewMySQLWebhookSubscriptionRepository(db)
		webhookDelRepo = repository.NewMySQLWebhookDeliveryRepository(db)
		toolInvRepo = repository.NewMySQLToolInvocationRepository(db)
		bindingRepo = repository.NewMySQLDeviceBindingRepository(db)
		slog.Info("connected to MySQL database")

		if err := migrations.Run(db); err != nil {
			slog.Error("auto-migration failed", "error", err)
			os.Exit(1)
		}
	} else {
		slog.Info("ENX_DATABASE_DSN not set, using in-memory tenant repo (limited functionality)")
		tenantRepo = repository.NewMemoryTenantRepository()
	}

	// --- Redis ---
	var redisClient *infrastructure.RedisClient
	if redisAddr != "" {
		var err error
		redisClient, err = infrastructure.NewRedisClient(redisAddr, redisPassword, 0)
		if err != nil {
			slog.Warn("Redis connection failed, running without cache", "error", err)
		}
	}

	// --- MinIO ---
	var minioClient *infrastructure.MinIOClient
	if minioEndpoint != "" {
		var err error
		minioClient, err = infrastructure.NewMinIOClient(minioEndpoint, minioAccessKey, minioSecretKey, minioBucket, false)
		if err != nil {
			slog.Warn("MinIO connection failed, running without object storage", "error", err)
		}
		if minioClient != nil && minioPublicEndpoint != "" {
			minioClient.SetPublicEndpoint(minioPublicEndpoint, minioAccessKey, minioSecretKey, false)
		}
	}

	// --- Services ---
	authService := auth.NewService(userRepo, jwtSecret, deviceSecret, sessionSecret)
	tenantService := tenant.NewService(tenantRepo)
	enrollService := enrollment.NewService(enrollRepo, deviceRepo, authService)
	auditService := audit.NewService(auditRepo)
	pkgService := package_svc.NewService(pkgRepo, minioClient)
	bindingService := device_binding.NewService(bindingRepo, pkgRepo)
	modelProfileService := model_profile.NewService(modelProfileRepo)
	policyProfileService := policy_profile.NewService(policyProfileRepo)
	agentProfileService := agent_profile.NewService(agentProfileRepo)
	deviceService := device.NewService(deviceRepo, authService)
	gatewayClient := infrastructure.NewGatewayClient(gatewayURL)
	if redisClient != nil {
		gatewayClient.SetRedisClient(redisClient)
	}
	sessionService := session.NewService(sessionRepo, approvalRepo, deviceRepo, auditRepo, authService, gatewayClient)
	if toolInvRepo != nil {
		sessionService.SetToolInvocationRepository(toolInvRepo)
	}
	rbacService := rbac.NewService(roleRepo, rbindingRepo)
	webhookService := webhook.NewService(webhookSubRepo, webhookDelRepo)
	var metricsService *metrics.Service
	var licenseService *license.Service
	if gormDB != nil {
		metricsService = metrics.NewService(repository.NewMySQLMetricsRepository(gormDB))
		licenseService = license.NewService(repository.NewMySQLLicenseRepository(gormDB))
		// Seed default roles for system tenant (fire and forget)
		go rbacService.SeedDefaultRoles(context.Background(), "system")
	}

	// --- Feishu Integration ---
	var feishuHandler *feishu.Handler
	feishuClient := feishu.NewClient(feishuAppID, feishuAppSecret)
	if feishuClient.Enabled() {
		var bridgeStore feishu.RedisStore
		if redisClient != nil {
			bridgeStore = redisClient
		}
		feishuBridge := feishu.NewChatBridge(bridgeStore)
		feishuBot := feishu.NewBotService(feishuClient, feishuBridge, sessionService)
		feishuSink := feishu.NewEventSink(feishuClient, feishuBridge)
		feishu.RegisterDefaultCommands(feishuBot, feishuBridge, deviceRepo, sessionService, auditRepo)
		feishuHandler = feishu.NewHandler(feishuBot, feishuSink, feishuVerifyToken)
		slog.Info("Feishu conversational integration enabled")
	}

	// --- Handlers ---
	tenantHandler := httphandler.NewTenantHandler(tenantService)
	tokenHandler := httphandler.NewTokenHandler(enrollService)
	pkgHandler := httphandler.NewPackageHandler(pkgService, bindingService)
	authHandler := httphandler.NewAuthHandler(authService)
	modelProfileHandler := httphandler.NewModelProfileHandler(modelProfileService)
	policyProfileHandler := httphandler.NewPolicyProfileHandler(policyProfileService)
	agentProfileHandler := httphandler.NewAgentProfileHandler(agentProfileService)
	deviceHandler := httphandler.NewDeviceHandler(deviceService)
	sessionHandler := httphandler.NewSessionHandler(sessionService)
	auditHandler := httphandler.NewAuditHandler(auditService)
	rbacHandler := httphandler.NewRBACHandler(rbacService)
	webhookHandler := httphandler.NewWebhookHandler(webhookService)
	var metricsHandler *httphandler.MetricsHandler
	var licenseHandler *httphandler.LicenseHandler
	if metricsService != nil {
		metricsHandler = httphandler.NewMetricsHandler(metricsService)
	}
	if licenseService != nil {
		licenseHandler = httphandler.NewLicenseHandler(licenseService)
	}

	agentEnrollHandler := agent.NewEnrollHandler(enrollService)
	agentAuditHandler := agent.NewAuditHandler(auditService)
	agentLifecycleHandler := agent.NewLifecycleHandler(deviceService, agentProfileRepo, modelProfileRepo, policyProfileRepo)
	agentApprovalHandler := agent.NewApprovalHandler(sessionService)
	agentActivateHandler := agent.NewActivateHandler(bindingService)

	// --- Router ---
	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     corsOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	router.Use(middleware.RateLimiter(50, 100))

	router.GET("/healthz", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})
	router.GET("/readyz", func(c *gin.Context) {
		checks := gin.H{}
		allOK := true

		if gormDB != nil {
			sqlDB, err := gormDB.DB()
			if err == nil {
				err = sqlDB.Ping()
			}
			checks["database"] = err == nil
			if err != nil {
				allOK = false
			}
		} else {
			checks["database"] = false
			allOK = false
		}

		if redisClient != nil {
			err := redisClient.Ping(c.Request.Context())
			checks["redis"] = err == nil
			if err != nil {
				allOK = false
			}
		} else {
			checks["redis"] = false
		}

		if minioClient != nil {
			err := minioClient.Ping(c.Request.Context())
			checks["minio"] = err == nil
			if err != nil {
				allOK = false
			}
		} else {
			checks["minio"] = false
		}

		if allOK {
			c.JSON(http.StatusOK, gin.H{"status": "ready", "checks": checks})
		} else {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready", "checks": checks})
		}
	})

	// Public endpoints (no auth required)
	publicV1 := router.Group("/api/v1")
	{
		publicV1.POST("/auth/login", middleware.RateLimiter(10, 20), func(c *gin.Context) { authHandler.Login(c) })
		publicV1.POST("/auth/refresh", middleware.RateLimiter(20, 40), func(c *gin.Context) { authHandler.RefreshToken(c) })
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
		rbacHandler.RegisterRoutes(protectedV1)
		webhookHandler.RegisterRoutes(protectedV1)
		if metricsHandler != nil {
			metricsHandler.RegisterRoutes(protectedV1)
		}
		if licenseHandler != nil {
			licenseHandler.RegisterRoutes(protectedV1)
		}
	}

	// Feishu webhook endpoints (no console auth, verified by Feishu token)
	if feishuHandler != nil {
		feishuGroup := router.Group("/webhook")
		feishuHandler.RegisterRoutes(feishuGroup)
	}

	// Agent API (device token or open for enrollment/activation)
	agentEnrollHandler.RegisterRoutes(router.Group(""))
	agentActivationGroup := router.Group("")
	agentActivationGroup.Use(middleware.RateLimiter(10, 20))
	agentActivateHandler.RegisterRoutes(agentActivationGroup)
	agentDeviceGroup := router.Group("")
	agentDeviceGroup.Use(middleware.DeviceAuth(deviceSecret))
	agentAuditHandler.RegisterRoutes(agentDeviceGroup)
	agentLifecycleHandler.RegisterRoutes(agentDeviceGroup)
	agentApprovalHandler.RegisterRoutes(agentDeviceGroup)

	// --- Server ---
	server := &http.Server{
		Addr:         ":" + httpPort,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		slog.Info("starting platform-api", "addr", ":"+httpPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
		os.Exit(1)
	}
	slog.Info("server exited")
}
