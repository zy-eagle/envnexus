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
	"github.com/zy-eagle/envnexus/services/job-runner/internal/worker"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	dsn := os.Getenv("ENX_DATABASE_DSN")
	healthPort := envOrDefault("ENX_HEALTH_PORT", "8082")

	var db *gorm.DB
	if dsn != "" {
		var err error
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}
		log.Println("Connected to MySQL database")
	} else {
		log.Println("Warning: ENX_DATABASE_DSN not set, workers requiring DB will be disabled")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if db != nil {
		go worker.NewTokenCleanupWorker(db).Start(ctx)
		go worker.NewLinkCleanupWorker(db).Start(ctx)
		go worker.NewAuditFlushWorker(db).Start(ctx)
		go worker.NewSessionCleanupWorker(db).Start(ctx)
	}

	router := gin.Default()
	router.GET("/healthz", func(c *gin.Context) {
		status := "ok"
		if db == nil {
			status = "degraded (no database)"
		}
		c.JSON(http.StatusOK, gin.H{
			"status":  status,
			"workers": []string{"token_cleanup", "link_cleanup", "audit_flush", "session_cleanup"},
		})
	})

	server := &http.Server{
		Addr:    ":" + healthPort,
		Handler: router,
	}

	go func() {
		log.Printf("Starting job-runner health server on :%s", healthPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down job-runner...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("job-runner exited")
}
