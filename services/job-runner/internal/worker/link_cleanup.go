package worker

import (
	"context"
	"log/slog"
	"time"

	"gorm.io/gorm"
)

type LinkCleanupWorker struct {
	db       *gorm.DB
	interval time.Duration
}

func NewLinkCleanupWorker(db *gorm.DB) *LinkCleanupWorker {
	return &LinkCleanupWorker{db: db, interval: 6 * time.Hour}
}

func (w *LinkCleanupWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	slog.Info("Worker started", "worker", "link_cleanup")

	for {
		select {
		case <-ctx.Done():
			slog.Info("Worker stopped", "worker", "link_cleanup")
			return
		case <-ticker.C:
			w.cleanup(ctx)
		}
	}
}

func (w *LinkCleanupWorker) cleanup(ctx context.Context) {
	cutoff := time.Now().Add(-30 * 24 * time.Hour)
	result := w.db.WithContext(ctx).Exec(
		`UPDATE download_packages SET status = 'expired' WHERE status = 'pending' AND created_at < ?`,
		cutoff,
	)
	if result.Error != nil {
		slog.Error("Link cleanup failed", "worker", "link_cleanup", "error", result.Error)
		return
	}
	if result.RowsAffected > 0 {
		slog.Info("Expired old packages", "worker", "link_cleanup", "count", result.RowsAffected)
	}
}
