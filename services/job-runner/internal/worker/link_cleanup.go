package worker

import (
	"context"
	"log"
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

	log.Println("[link_cleanup] Worker started")

	for {
		select {
		case <-ctx.Done():
			log.Println("[link_cleanup] Worker stopped")
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
		log.Printf("[link_cleanup] Error: %v\n", result.Error)
		return
	}
	if result.RowsAffected > 0 {
		log.Printf("[link_cleanup] Expired %d old packages\n", result.RowsAffected)
	}
}
