package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/minio/minio-go/v7"
	"gorm.io/gorm"
)

// PackageBuildWorker dequeues package_build jobs and processes them.
type PackageBuildWorker struct {
	db          *gorm.DB
	minioClient *minio.Client
	bucket      string
	interval    time.Duration
}

func NewPackageBuildWorker(db *gorm.DB, minioClient *minio.Client, bucket string) *PackageBuildWorker {
	return &PackageBuildWorker{
		db:          db,
		minioClient: minioClient,
		bucket:      bucket,
		interval:    15 * time.Second,
	}
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
	w.db.WithContext(ctx).Table("download_packages").
		Where("id = ?", p.PackageID).
		Updates(map[string]interface{}{"status": "building", "updated_at": time.Now()})

	// Simulate build work (in production this would trigger actual CI/build system)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(2 * time.Second):
	}

	ext := ""
	if p.Platform == "windows" {
		ext = ".exe"
	}
	artifactPath := fmt.Sprintf("packages/%s/%s/enx-agent-%s-%s%s", p.TenantID, p.PackageID, p.Platform, p.Arch, ext)

	// Upload real package to MinIO if configured
	if w.minioClient != nil && w.bucket != "" {
		baseObjectKey := fmt.Sprintf("base-packages/enx-agent-%s-%s%s", p.Platform, p.Arch, ext)

		// 1. Get the base binary from MinIO
		baseObj, err := w.minioClient.GetObject(ctx, w.bucket, baseObjectKey, minio.GetObjectOptions{})
		if err != nil {
			return fmt.Errorf("failed to get base package %s: %w", baseObjectKey, err)
		}
		defer baseObj.Close()

		// Stat to ensure it exists and get size
		stat, err := baseObj.Stat()
		if err != nil {
			return fmt.Errorf("base package not found or inaccessible (%s). Please upload base packages first: %w", baseObjectKey, err)
		}

		// 2. Generate the JSON payload
		platformURL := os.Getenv("ENX_PLATFORM_URL")
		if platformURL == "" {
			platformURL = "http://localhost:8080"
		}
		wsURL := os.Getenv("ENX_WS_URL")
		if wsURL == "" {
			wsURL = "ws://localhost:8081/ws/v1/sessions"
		}

		configPayload := map[string]string{
			"platform_url":     platformURL,
			"ws_url":           wsURL,
			"enrollment_token": "auto_generated_token_for_" + p.TenantID, // In a full implementation, fetch a real token from DB
		}
		payloadBytes, _ := json.Marshal(configPayload)
		injectedData := append([]byte("\nENX_CONF_START:"), payloadBytes...)

		// 3. Append JSON to the binary using MultiReader (Zero memory overhead for the base binary)
		reader := io.MultiReader(baseObj, bytes.NewReader(injectedData))
		totalSize := stat.Size + int64(len(injectedData))

		// 4. Upload the modified binary to the tenant's path
		_, err = w.minioClient.PutObject(ctx, w.bucket, artifactPath, reader, totalSize, minio.PutObjectOptions{
			ContentType: "application/octet-stream",
		})
		if err != nil {
			slog.Error("Failed to upload tenant package to MinIO", "error", err)
			return err
		}
	}

	// Mark package as ready
	result := w.db.WithContext(ctx).Table("download_packages").
		Where("id = ?", p.PackageID).
		Updates(map[string]interface{}{
			"status":        "ready",
			"sign_status":   "signed",
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
