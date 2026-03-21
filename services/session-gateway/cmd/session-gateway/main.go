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
	"github.com/zy-eagle/envnexus/services/session-gateway/internal/handler/ws"
)

func main() {
	// 1. Load config and env vars
	// 2. Init logger and Redis connection
	// 3. Init repository, service, handler, middleware
	sessionManager := ws.NewSessionManager()

	router := gin.Default()
	
	// Health checks
	router.GET("/healthz", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})
	router.GET("/readyz", func(c *gin.Context) {
		c.String(http.StatusOK, "Ready")
	})

	// 4. Register WebSocket routes
	router.GET("/ws/v1/agent", sessionManager.HandleAgentConnection)
	// Internal API for platform-api to send commands to agents
	router.POST("/api/v1/sessions/command", sessionManager.HandleCommand)

	server := &http.Server{
		Addr:    ":8081",
		Handler: router,
	}

	go func() {
		log.Println("Starting session-gateway server on :8081")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
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
