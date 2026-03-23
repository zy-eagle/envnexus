package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/zy-eagle/envnexus/apps/agent-core/internal/bootstrap"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))
	slog.Info("Starting enx-agent core...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Run bootstrap sequence
	bootstrapper := bootstrap.NewBootstrapper()
	if err := bootstrapper.Run(ctx); err != nil {
		slog.Error("Bootstrap failed", "error", err)
		os.Exit(1)
	}

	// Wait for interrupt signal to gracefully shutdown the agent
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down enx-agent...")
	cancel()

	slog.Info("enx-agent exiting")
}
