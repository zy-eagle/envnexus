package worker

import (
	"context"
	"log/slog"
	"time"

	"gorm.io/gorm"
)

type SessionCleanupWorker struct {
	db       *gorm.DB
	interval time.Duration
}

func NewSessionCleanupWorker(db *gorm.DB) *SessionCleanupWorker {
	return &SessionCleanupWorker{db: db, interval: 2 * time.Hour}
}

func (w *SessionCleanupWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	slog.Info("Worker started", "worker", "session_cleanup")

	for {
		select {
		case <-ctx.Done():
			slog.Info("Worker stopped", "worker", "session_cleanup")
			return
		case <-ticker.C:
			w.cleanup(ctx)
		}
	}
}

func (w *SessionCleanupWorker) cleanup(ctx context.Context) {
	expireCutoff := time.Now().Add(-2 * time.Hour)

	result := w.db.WithContext(ctx).Exec(
		`UPDATE sessions SET status = 'expired', ended_at = NOW(3), updated_at = NOW(3) WHERE status IN ('created', 'attached') AND started_at < ?`,
		expireCutoff,
	)
	if result.Error != nil {
		slog.Error("Session cleanup failed", "worker", "session_cleanup", "error", result.Error)
		return
	}
	if result.RowsAffected > 0 {
		slog.Info("Expired stale sessions", "worker", "session_cleanup", "count", result.RowsAffected)
	}

	approvalResult := w.db.WithContext(ctx).Exec(
		`UPDATE approval_requests SET status = 'expired', updated_at = NOW(3) WHERE status = 'pending_user' AND expires_at IS NOT NULL AND expires_at < NOW(3)`,
	)
	if approvalResult.Error != nil {
		slog.Error("Approval request expire failed", "worker", "session_cleanup", "error", approvalResult.Error)
		return
	}
	if approvalResult.RowsAffected > 0 {
		slog.Info("Expired approval requests", "worker", "session_cleanup", "count", approvalResult.RowsAffected)
	}
}
