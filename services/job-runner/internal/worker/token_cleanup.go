package worker

import (
	"context"
	"log"
	"time"

	"gorm.io/gorm"
)

type TokenCleanupWorker struct {
	db *gorm.DB
}

func NewTokenCleanupWorker(db *gorm.DB) *TokenCleanupWorker {
	return &TokenCleanupWorker{db: db}
}

func (w *TokenCleanupWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	log.Println("Starting TokenCleanupWorker...")

	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping TokenCleanupWorker...")
			return
		case <-ticker.C:
			w.cleanup(ctx)
		}
	}
}

func (w *TokenCleanupWorker) cleanup(ctx context.Context) {
	log.Println("Running token cleanup job...")
	
	// Delete tokens that are expired or have reached max uses
	result := w.db.WithContext(ctx).Exec(`
		DELETE FROM enrollment_tokens 
		WHERE expires_at < ? OR used_count >= max_uses
	`, time.Now())

	if result.Error != nil {
		log.Printf("Failed to cleanup tokens: %v\n", result.Error)
		return
	}

	log.Printf("Cleaned up %d expired/exhausted tokens\n", result.RowsAffected)
}
