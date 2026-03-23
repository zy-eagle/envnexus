package worker

import (
	"context"
	"log/slog"
	"time"

	"gorm.io/gorm"
)

type TokenCleanupWorker struct {
	db       *gorm.DB
	interval time.Duration
}

func NewTokenCleanupWorker(db *gorm.DB) *TokenCleanupWorker {
	return &TokenCleanupWorker{db: db, interval: 1 * time.Hour}
}

func (w *TokenCleanupWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	slog.Info("Worker started", "worker", "token_cleanup")
	w.cleanup(ctx)

	for {
		select {
		case <-ctx.Done():
			slog.Info("Worker stopped", "worker", "token_cleanup")
			return
		case <-ticker.C:
			w.cleanup(ctx)
		}
	}
}

func (w *TokenCleanupWorker) cleanup(ctx context.Context) {
	result := w.db.WithContext(ctx).Exec(
		`UPDATE enrollment_tokens SET status = 'expired' WHERE status = 'active' AND (expires_at < ? OR used_count >= max_uses)`,
		time.Now(),
	)
	if result.Error != nil {
		slog.Error("Token cleanup failed", "worker", "token_cleanup", "error", result.Error)
		return
	}
	if result.RowsAffected > 0 {
		slog.Info("Expired tokens", "worker", "token_cleanup", "count", result.RowsAffected)
	}
}
