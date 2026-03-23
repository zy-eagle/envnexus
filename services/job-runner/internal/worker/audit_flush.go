package worker

import (
	"context"
	"log/slog"
	"time"

	"gorm.io/gorm"
)

type AuditFlushWorker struct {
	db       *gorm.DB
	interval time.Duration
}

func NewAuditFlushWorker(db *gorm.DB) *AuditFlushWorker {
	return &AuditFlushWorker{db: db, interval: 30 * time.Minute}
}

func (w *AuditFlushWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	slog.Info("Worker started", "worker", "audit_flush")

	for {
		select {
		case <-ctx.Done():
			slog.Info("Worker stopped", "worker", "audit_flush")
			return
		case <-ticker.C:
			w.flush(ctx)
		}
	}
}

func (w *AuditFlushWorker) flush(ctx context.Context) {
	var count int64
	w.db.WithContext(ctx).Raw("SELECT COUNT(*) FROM audit_events WHERE created_at > ?",
		time.Now().Add(-24*time.Hour)).Scan(&count)
	slog.Info("Recent audit events counted", "worker", "audit_flush", "window_hours", 24, "count", count)
}
