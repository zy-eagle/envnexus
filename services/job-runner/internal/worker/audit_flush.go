package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/minio/minio-go/v7"
	"gorm.io/gorm"
)

type auditEvent struct {
	ID               string  `json:"id"`
	TenantID         string  `json:"tenant_id"`
	DeviceID         *string `json:"device_id"`
	SessionID        *string `json:"session_id"`
	EventType        string  `json:"event_type"`
	EventPayloadJSON string  `json:"event_payload_json"`
	CreatedAt        string  `json:"created_at"`
}

type AuditFlushWorker struct {
	db            *gorm.DB
	minioClient   *minio.Client
	bucketName    string
	interval      time.Duration
	retentionDays int
	batchSize     int
}

func NewAuditFlushWorker(db *gorm.DB, minioClient *minio.Client, bucketName string) *AuditFlushWorker {
	return &AuditFlushWorker{
		db:            db,
		minioClient:   minioClient,
		bucketName:    bucketName,
		interval:      30 * time.Minute,
		retentionDays: 30,
		batchSize:     1000,
	}
}

func (w *AuditFlushWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	slog.Info("Worker started", "worker", "audit_flush", "retention_days", w.retentionDays)

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
	cutoff := time.Now().Add(-time.Duration(w.retentionDays) * 24 * time.Hour)

	var events []auditEvent
	result := w.db.WithContext(ctx).
		Table("audit_events").
		Where("created_at < ? AND (archived = 0 OR archived IS NULL)", cutoff).
		Order("created_at ASC").
		Limit(w.batchSize).
		Find(&events)

	if result.Error != nil {
		slog.Error("Failed to query audit events for archival", "worker", "audit_flush", "error", result.Error)
		return
	}

	if len(events) == 0 {
		slog.Info("No audit events to archive", "worker", "audit_flush", "cutoff", cutoff.Format(time.RFC3339))
		return
	}

	slog.Info("Archiving audit events", "worker", "audit_flush", "count", len(events), "cutoff", cutoff.Format(time.RFC3339))

	if w.minioClient != nil {
		if err := w.uploadToMinIO(ctx, events); err != nil {
			slog.Warn("MinIO upload failed, falling back to local filesystem archive", "worker", "audit_flush", "error", err)
			if err2 := w.writeToLocalFS(events); err2 != nil {
				slog.Error("Local filesystem fallback also failed", "worker", "audit_flush", "error", err2)
				return
			}
		}
	} else {
		// Phase 5: offline fallback — write to local filesystem
		slog.Info("MinIO not configured, writing audit archive to local filesystem", "worker", "audit_flush")
		if err := w.writeToLocalFS(events); err != nil {
			slog.Error("Failed to write audit archive to local filesystem", "worker", "audit_flush", "error", err)
			return
		}
	}

	ids := make([]string, len(events))
	for i, e := range events {
		ids[i] = e.ID
	}

	markResult := w.db.WithContext(ctx).
		Table("audit_events").
		Where("id IN ?", ids).
		Update("archived", true)

	if markResult.Error != nil {
		slog.Error("Failed to mark audit events as archived", "worker", "audit_flush", "error", markResult.Error)
		return
	}

	slog.Info("Audit events archived successfully", "worker", "audit_flush", "archived_count", markResult.RowsAffected)
}

func (w *AuditFlushWorker) uploadToMinIO(ctx context.Context, events []auditEvent) error {
	data, err := json.Marshal(events)
	if err != nil {
		return fmt.Errorf("marshal events: %w", err)
	}

	objectName := fmt.Sprintf("audit-archives/%s/audit_%s.json",
		time.Now().Format("2006/01/02"),
		time.Now().Format("20060102_150405"))

	reader := bytes.NewReader(data)
	_, err = w.minioClient.PutObject(ctx, w.bucketName, objectName, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: "application/json",
	})
	if err != nil {
		return fmt.Errorf("upload to MinIO: %w", err)
	}

	slog.Info("Uploaded audit archive to MinIO", "worker", "audit_flush", "object", objectName, "size_bytes", len(data))
	return nil
}

// writeToLocalFS is the Phase 5 offline fallback — saves audit archives to local disk.
func (w *AuditFlushWorker) writeToLocalFS(events []auditEvent) error {
	archiveDir := "/var/lib/envnexus/audit-archives"
	if v := os.Getenv("ENX_AUDIT_ARCHIVE_DIR"); v != "" {
		archiveDir = v
	}

	dateDir := filepath.Join(archiveDir, time.Now().Format("2006/01/02"))
	if err := os.MkdirAll(dateDir, 0750); err != nil {
		return fmt.Errorf("create archive dir %s: %w", dateDir, err)
	}

	filename := filepath.Join(dateDir, fmt.Sprintf("audit_%s.json", time.Now().Format("20060102_150405")))
	data, err := json.Marshal(events)
	if err != nil {
		return fmt.Errorf("marshal events: %w", err)
	}

	if err := os.WriteFile(filename, data, 0640); err != nil {
		return fmt.Errorf("write audit archive %s: %w", filename, err)
	}

	slog.Info("Wrote audit archive to local filesystem", "worker", "audit_flush", "file", filename, "size_bytes", len(data))
	return nil
}
