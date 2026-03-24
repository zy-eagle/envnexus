package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zy-eagle/envnexus/apps/agent-core/internal/bootstrap"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))
	slog.Info("Starting enx-agent core...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bootstrapper := bootstrap.NewBootstrapper()
	if err := bootstrapper.Run(ctx); err != nil {
		slog.Error("Bootstrap failed", "error", err)
		os.Exit(1)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down enx-agent...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	bootstrapper.Shutdown(shutdownCtx)

	slog.Info("enx-agent exiting")
}
