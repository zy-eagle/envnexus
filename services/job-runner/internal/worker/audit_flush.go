package worker

import (
	"context"
	"log"
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

	log.Println("[audit_flush] Worker started")

	for {
		select {
		case <-ctx.Done():
			log.Println("[audit_flush] Worker stopped")
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
	log.Printf("[audit_flush] Recent audit events (24h): %d\n", count)
}
