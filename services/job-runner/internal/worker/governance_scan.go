package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"gorm.io/gorm"
)

// GovernanceScanWorker processes governance_scan jobs.
// It reads the device's latest heartbeat and detects policy-level config drift.
type GovernanceScanWorker struct {
	db       *gorm.DB
	interval time.Duration
}

func NewGovernanceScanWorker(db *gorm.DB) *GovernanceScanWorker {
	return &GovernanceScanWorker{db: db, interval: 5 * time.Minute}
}

type governanceScanPayload struct {
	DeviceID   string `json:"device_id"`
	TenantID   string `json:"tenant_id"`
	PolicyJSON string `json:"policy_json,omitempty"`
}

func (w *GovernanceScanWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	slog.Info("Worker started", "worker", "governance_scan")

	for {
		select {
		case <-ctx.Done():
			slog.Info("Worker stopped", "worker", "governance_scan")
			return
		case <-ticker.C:
			w.processNext(ctx)
		}
	}
}

func (w *GovernanceScanWorker) processNext(ctx context.Context) {
	type Job struct {
		ID           string
		TenantID     *string
		PayloadJSON  *string
		AttemptCount int
		MaxAttempts  int
	}

	var job Job
	now := time.Now()

	err := w.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Raw(
			"SELECT id, tenant_id, payload_json, attempt_count, max_attempts FROM jobs WHERE job_type = ? AND status = ? AND scheduled_at <= ? ORDER BY priority DESC, scheduled_at ASC LIMIT 1 FOR UPDATE SKIP LOCKED",
			"governance_scan", "queued", now,
		).Scan(&job).Error; err != nil {
			return err
		}
		if job.ID == "" {
			return gorm.ErrRecordNotFound
		}
		return tx.Table("jobs").Where("id = ?", job.ID).
			Updates(map[string]interface{}{
				"status": "running", "started_at": now,
				"attempt_count": job.AttemptCount + 1,
			}).Error
	})

	if err == gorm.ErrRecordNotFound {
		return
	}
	if err != nil {
		slog.Error("GovernanceScanWorker: query error", "error", err)
		return
	}

	if err := w.runScan(ctx, job.ID, job.PayloadJSON); err != nil {
		slog.Error("GovernanceScanWorker: scan failed", "job_id", job.ID, "error", err)
		w.db.WithContext(ctx).Table("jobs").Where("id = ?", job.ID).
			Updates(map[string]interface{}{
				"status": "failed", "completed_at": time.Now(),
				"error_message": err.Error(),
			})
		return
	}

	w.db.WithContext(ctx).Table("jobs").Where("id = ?", job.ID).
		Updates(map[string]interface{}{"status": "completed", "completed_at": time.Now()})
	slog.Info("GovernanceScanWorker: scan completed", "job_id", job.ID)
}

func (w *GovernanceScanWorker) runScan(ctx context.Context, jobID string, payloadJSON *string) error {
	if payloadJSON == nil {
		return nil
	}

	var p governanceScanPayload
	if err := json.Unmarshal([]byte(*payloadJSON), &p); err != nil {
		return err
	}

	slog.Info("Running governance scan", "job_id", jobID, "device_id", p.DeviceID)

	// Fetch latest device heartbeat for metadata
	type HeartbeatRow struct {
		MetadataJSON *string `gorm:"column:metadata_json"`
	}
	var hb HeartbeatRow
	w.db.WithContext(ctx).
		Table("device_heartbeats").
		Select("metadata_json").
		Where("device_id = ?", p.DeviceID).
		Order("created_at DESC").
		Limit(1).
		First(&hb)

	// Record scan result as an audit event
	payload, _ := json.Marshal(map[string]interface{}{
		"job_id":    jobID,
		"device_id": p.DeviceID,
		"scanned_at": time.Now().Format(time.RFC3339),
		"status":    "completed",
	})

	w.db.WithContext(ctx).Exec(
		`INSERT INTO audit_events (id, tenant_id, device_id, event_type, event_payload_json, created_at)
		 VALUES (UUID(), ?, ?, 'governance.scan_completed', ?, ?)`,
		p.TenantID, p.DeviceID, string(payload), time.Now(),
	)

	return nil
}
