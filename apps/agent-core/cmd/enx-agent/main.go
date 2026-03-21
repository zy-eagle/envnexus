package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/zy-eagle/envnexus/apps/agent-core/internal/bootstrap"
)

func main() {
	log.Println("Starting enx-agent core...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Run bootstrap sequence
	bootstrapper := bootstrap.NewBootstrapper()
	if err := bootstrapper.Run(ctx); err != nil {
		log.Fatalf("Bootstrap failed: %v", err)
	}

	// Wait for interrupt signal to gracefully shutdown the agent
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	
	log.Println("Shutting down enx-agent...")
	cancel()

	log.Println("enx-agent exiting")
}
