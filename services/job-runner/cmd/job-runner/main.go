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

func main() {
	// 1. Load config and env vars
	dsn := os.Getenv("ENX_DATABASE_DSN")
	
	// 2. Init logger, DB, Redis, Object Storage
	var db *gorm.DB
	if dsn != "" {
		var err error
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}
		log.Println("Successfully connected to MySQL database")
	} else {
		log.Println("Warning: ENX_DATABASE_DSN not set, workers requiring DB will fail")
	}

	// 3. Init repository, service, worker
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 4. Start task consumers
	if db != nil {
		tokenWorker := worker.NewTokenCleanupWorker(db)
		go tokenWorker.Start(ctx)
	}
	// 5. Start health check HTTP server
	router := gin.Default()
	
	// Health checks
	router.GET("/healthz", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	server := &http.Server{
		Addr:    ":8082", // Internal port for health checks
		Handler: router,
	}

	go func() {
		log.Println("Starting job-runner health server on :8082")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// 6. Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down job-runner...")

	// Use a new variable name for the shutdown context to avoid shadowing/redeclaring
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("job-runner exiting")
}
