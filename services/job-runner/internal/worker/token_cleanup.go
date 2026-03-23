package worker

import (
	"context"
	"log"
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

	log.Println("[token_cleanup] Worker started")
	w.cleanup(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("[token_cleanup] Worker stopped")
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
		log.Printf("[token_cleanup] Error: %v\n", result.Error)
		return
	}
	if result.RowsAffected > 0 {
		log.Printf("[token_cleanup] Expired %d tokens\n", result.RowsAffected)
	}
}
