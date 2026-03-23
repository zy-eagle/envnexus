package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zy-eagle/envnexus/services/session-gateway/internal/handler/ws"
)

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	port := envOrDefault("ENX_HTTP_PORT", "8081")
	redisAddr := envOrDefault("ENX_REDIS_ADDR", "localhost:6379")
	redisPassword := os.Getenv("ENX_REDIS_PASSWORD")
	sessionTokenSecret := envOrDefault("ENX_SESSION_TOKEN_SECRET", "dev-session-secret-change-me")

	sessionManager := ws.NewSessionManager(sessionTokenSecret)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rc, err := ws.NewRedisClient(redisAddr, redisPassword, 0)
	if err != nil {
		log.Printf("Warning: Redis connection failed (%s): %v. Running without pub/sub.", redisAddr, err)
	} else {
		sessionManager.SetRedisClient(rc)
		go rc.SubscribeSessionEvents(ctx)
		log.Printf("Connected to Redis at %s", redisAddr)
	}

	router := gin.Default()

	router.GET("/healthz", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})
	router.GET("/readyz", func(c *gin.Context) {
		status := "ready"
		redisOK := false
		if rc != nil {
			if err := rc.Ping(c.Request.Context()); err == nil {
				redisOK = true
			}
		}
		c.JSON(http.StatusOK, gin.H{
			"status":         status,
			"online_devices": sessionManager.GetOnlineDeviceCount(),
			"redis":          redisOK,
		})
	})

	router.GET("/ws/v1/sessions/:deviceId", sessionManager.HandleAgentConnection)
	router.POST("/api/v1/sessions/command", sessionManager.HandleCommand)
	router.POST("/api/v1/sessions/:sessionId/events", sessionManager.HandleSessionEvent)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		log.Printf("Starting session-gateway on :%s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down session-gateway...")

	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if rc != nil {
		rc.Close()
	}

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Session-gateway exited")
}
