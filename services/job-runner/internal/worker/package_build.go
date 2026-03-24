package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"gorm.io/gorm"
)

// PackageBuildWorker dequeues package_build jobs and processes them.
type PackageBuildWorker struct {
	db       *gorm.DB
	interval time.Duration
}

func NewPackageBuildWorker(db *gorm.DB) *PackageBuildWorker {
	return &PackageBuildWorker{db: db, interval: 15 * time.Second}
}

type packageBuildPayload struct {
	PackageID string `json:"package_id"`
	TenantID  string `json:"tenant_id"`
	Platform  string `json:"platform"`
	Arch      string `json:"arch"`
}

func (w *PackageBuildWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	slog.Info("Worker started", "worker", "package_build")

	for {
		select {
		case <-ctx.Done():
			slog.Info("Worker stopped", "worker", "package_build")
			return
		case <-ticker.C:
			w.processNext(ctx)
		}
	}
}

func (w *PackageBuildWorker) processNext(ctx context.Context) {
	type Job struct {
		ID           string
		PayloadJSON  *string
		AttemptCount int
		MaxAttempts  int
	}

	var job Job
	now := time.Now()

	err := w.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Raw(
			"SELECT id, payload_json, attempt_count, max_attempts FROM jobs WHERE job_type = ? AND status = ? AND scheduled_at <= ? ORDER BY priority DESC, scheduled_at ASC LIMIT 1 FOR UPDATE SKIP LOCKED",
			"package_build", "queued", now,
		).Scan(&job).Error; err != nil {
			return err
		}
		if job.ID == "" {
			return gorm.ErrRecordNotFound
		}
		return tx.Table("jobs").Where("id = ?", job.ID).
			Updates(map[string]interface{}{"status": "running", "started_at": now, "attempt_count": job.AttemptCount + 1}).Error
	})

	if err == gorm.ErrRecordNotFound {
		return
	}
	if err != nil {
		slog.Error("PackageBuildWorker: query error", "error", err)
		return
	}

	if err := w.buildPackage(ctx, job.ID, job.PayloadJSON); err != nil {
		slog.Error("PackageBuildWorker: build failed", "job_id", job.ID, "error", err)

		status := "failed"
		if job.AttemptCount+1 < job.MaxAttempts {
			// Retry with backoff
			retryAt := time.Now().Add(time.Duration(job.AttemptCount+1) * 5 * time.Minute)
			status = "queued"
			w.db.WithContext(ctx).Table("jobs").Where("id = ?", job.ID).
				Updates(map[string]interface{}{
					"status": status, "scheduled_at": retryAt,
					"error_message": err.Error(), "completed_at": nil,
				})
		} else {
			w.db.WithContext(ctx).Table("jobs").Where("id = ?", job.ID).
				Updates(map[string]interface{}{
					"status": status, "completed_at": time.Now(),
					"error_message": err.Error(),
				})
		}
		return
	}

	w.db.WithContext(ctx).Table("jobs").Where("id = ?", job.ID).
		Updates(map[string]interface{}{"status": "completed", "completed_at": time.Now()})
	slog.Info("PackageBuildWorker: job completed", "job_id", job.ID)
}

func (w *PackageBuildWorker) buildPackage(ctx context.Context, jobID string, payloadJSON *string) error {
	if payloadJSON == nil {
		return nil
	}
	var p packageBuildPayload
	if err := json.Unmarshal([]byte(*payloadJSON), &p); err != nil {
		return err
	}

	slog.Info("Building package",
		"job_id", jobID,
		"package_id", p.PackageID,
		"platform", p.Platform,
		"arch", p.Arch,
	)

	// Update package status to building
	w.db.WithContext(ctx).Table("packages").
		Where("id = ?", p.PackageID).
		Updates(map[string]interface{}{"status": "building", "updated_at": time.Now()})

	// Simulate build work (in production this would trigger actual CI/build system)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(2 * time.Second):
	}

	artifactPath := "packages/" + p.TenantID + "/" + p.PackageID + "/enx-agent-" + p.Platform + "-" + p.Arch

	// Mark package as ready
	result := w.db.WithContext(ctx).Table("packages").
		Where("id = ?", p.PackageID).
		Updates(map[string]interface{}{
			"status":        "ready",
			"artifact_path": artifactPath,
			"updated_at":    time.Now(),
		})

	if result.Error != nil {
		return result.Error
	}

	slog.Info("Package built successfully",
		"package_id", p.PackageID,
		"artifact_path", artifactPath,
	)
	return nil
}
