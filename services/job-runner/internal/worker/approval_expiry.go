package worker

import (
	"context"
	"log/slog"
	"time"

	"gorm.io/gorm"
)

// ApprovalExpiryWorker expires pending approval requests that have passed their deadline.
type ApprovalExpiryWorker struct {
	db       *gorm.DB
	interval time.Duration
}

func NewApprovalExpiryWorker(db *gorm.DB) *ApprovalExpiryWorker {
	return &ApprovalExpiryWorker{db: db, interval: 1 * time.Minute}
}

func (w *ApprovalExpiryWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	slog.Info("Worker started", "worker", "approval_expiry")

	for {
		select {
		case <-ctx.Done():
			slog.Info("Worker stopped", "worker", "approval_expiry")
			return
		case <-ticker.C:
			w.expire(ctx)
		}
	}
}

func (w *ApprovalExpiryWorker) expire(ctx context.Context) {
	result := w.db.WithContext(ctx).
		Table("approval_requests").
		Where("status = ? AND expires_at < ? AND expires_at IS NOT NULL", "pending_user", time.Now()).
		Updates(map[string]interface{}{
			"status":     "expired",
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		slog.Error("ApprovalExpiryWorker: failed to expire approvals", "error", result.Error)
		return
	}
	if result.RowsAffected > 0 {
		slog.Info("Expired pending approvals", "worker", "approval_expiry", "count", result.RowsAffected)
	}
}
