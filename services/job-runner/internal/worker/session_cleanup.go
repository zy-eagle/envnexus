package worker

import (
	"context"
	"log"
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

	log.Println("[session_cleanup] Worker started")

	for {
		select {
		case <-ctx.Done():
			log.Println("[session_cleanup] Worker stopped")
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
		log.Printf("[session_cleanup] Error: %v\n", result.Error)
		return
	}
	if result.RowsAffected > 0 {
		log.Printf("[session_cleanup] Expired %d stale sessions\n", result.RowsAffected)
	}

	approvalResult := w.db.WithContext(ctx).Exec(
		`UPDATE approval_requests SET status = 'expired', updated_at = NOW(3) WHERE status = 'pending_user' AND expires_at IS NOT NULL AND expires_at < NOW(3)`,
	)
	if approvalResult.Error != nil {
		log.Printf("[session_cleanup] Approval expire error: %v\n", approvalResult.Error)
		return
	}
	if approvalResult.RowsAffected > 0 {
		log.Printf("[session_cleanup] Expired %d approval requests\n", approvalResult.RowsAffected)
	}
}
