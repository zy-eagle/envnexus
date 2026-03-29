package worker

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"gorm.io/gorm"
)

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
	PackageID      string `json:"package_id"`
	TenantID       string `json:"tenant_id"`
	Platform       string `json:"platform"`
	Arch           string `json:"arch"`
	ActivationMode string `json:"activation_mode,omitempty"`
	ActivationKey  string `json:"activation_key,omitempty"`
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
			// Also mark the download_package as failed
			if job.PayloadJSON != nil {
				var p packageBuildPayload
				if json.Unmarshal([]byte(*job.PayloadJSON), &p) == nil && p.PackageID != "" {
					w.db.WithContext(ctx).Table("download_packages").Where("id = ?", p.PackageID).
						Updates(map[string]interface{}{"status": "failed", "updated_at": time.Now()})
				}
			}
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

	w.db.WithContext(ctx).Table("download_packages").
		Where("id = ?", p.PackageID).
		Updates(map[string]interface{}{"status": "building", "updated_at": time.Now()})

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(2 * time.Second):
	}

	artifactPath := fmt.Sprintf("packages/%s/%s/EnvNexus-Agent-%s-%s.zip", p.TenantID, p.PackageID, p.Platform, p.Arch)

	if w.minioClient == nil || w.bucket == "" {
		return fmt.Errorf("MinIO not configured")
	}

	configENX := w.buildConfigENX(p)

	installerData, installerName, err := w.downloadInstaller(ctx, p.Platform, p.Arch)
	if err != nil {
		// Windows/macOS MUST have a desktop installer (with UI) — no fallback to CLI binary.
		// Only Linux is allowed to fall back to raw binary for headless server use.
		if p.Platform != "linux" {
			return fmt.Errorf("desktop installer not found for %s/%s — please run agent-builder first to produce the installer: %w", p.Platform, p.Arch, err)
		}
		slog.Warn("Installer not found for Linux, falling back to raw binary", "error", err)
		return w.buildFallbackPackage(ctx, p, configENX, artifactPath)
	}

	// Create ZIP: installer + agent.enx (only 2 files)
	outputZip, err := bundleInstallerWithConfig(installerData, installerName, configENX, p.Platform)
	if err != nil {
		return fmt.Errorf("failed to create installer bundle: %w", err)
	}

	_, err = w.minioClient.PutObject(ctx, w.bucket, artifactPath,
		bytes.NewReader(outputZip), int64(len(outputZip)),
		minio.PutObjectOptions{ContentType: "application/zip"},
	)
	if err != nil {
		return fmt.Errorf("failed to upload package: %w", err)
	}

	return w.markReady(ctx, p.PackageID, artifactPath)
}

// downloadInstaller tries to fetch the desktop installer from MinIO.
func (w *PackageBuildWorker) downloadInstaller(ctx context.Context, platform, arch string) ([]byte, string, error) {
	candidates := installerCandidates(platform, arch)
	for _, key := range candidates {
		obj, err := w.minioClient.GetObject(ctx, w.bucket, "base-packages/"+key, minio.GetObjectOptions{})
		if err != nil {
			continue
		}
		if _, err := obj.Stat(); err != nil {
			obj.Close()
			continue
		}
		data, err := io.ReadAll(obj)
		obj.Close()
		if err != nil {
			continue
		}
		slog.Info("Found installer", "key", key, "size", len(data))
		return data, key, nil
	}
	return nil, "", fmt.Errorf("no installer found for %s-%s, tried: %v", platform, arch, candidates)
}

func installerCandidates(platform, arch string) []string {
	switch platform {
	case "windows":
		return []string{
			fmt.Sprintf("EnvNexus-Agent-Setup-windows-%s.exe", arch),
			fmt.Sprintf("EnvNexus-Agent-Setup-%s.exe", arch),
			"EnvNexus-Agent-Setup-windows-amd64.exe",
		}
	case "linux":
		return []string{
			fmt.Sprintf("EnvNexus-Agent-linux-%s.AppImage", arch),
			fmt.Sprintf("EnvNexus-Agent-%s.AppImage", arch),
			"EnvNexus-Agent-linux-amd64.AppImage",
		}
	case "darwin":
		return []string{
			fmt.Sprintf("EnvNexus-Agent-darwin-%s.dmg", arch),
			fmt.Sprintf("EnvNexus-Agent-%s.dmg", arch),
		}
	}
	return nil
}

// bundleInstallerWithConfig creates a ZIP containing only the installer + agent.enx config.
func bundleInstallerWithConfig(installerData []byte, installerName string, configENX []byte, platform string) ([]byte, error) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	iw, err := w.Create(installerName)
	if err != nil {
		return nil, err
	}
	if _, err := iw.Write(installerData); err != nil {
		return nil, err
	}

	cw, err := w.Create("agent.enx")
	if err != nil {
		return nil, err
	}
	if _, err := cw.Write(configENX); err != nil {
		return nil, err
	}

	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// buildFallbackPackage creates a ZIP with raw binary + agent.enx when no installer is available.
func (w *PackageBuildWorker) buildFallbackPackage(ctx context.Context, p packageBuildPayload, configENX []byte, artifactPath string) error {
	ext := ""
	if p.Platform == "windows" {
		ext = ".exe"
	}
	binaryKey := fmt.Sprintf("base-packages/enx-agent-%s-%s%s", p.Platform, p.Arch, ext)

	obj, err := w.minioClient.GetObject(ctx, w.bucket, binaryKey, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to get base binary %s: %w", binaryKey, err)
	}
	defer obj.Close()

	if _, err := obj.Stat(); err != nil {
		return fmt.Errorf("base binary not found (%s): %w", binaryKey, err)
	}

	binaryData, err := io.ReadAll(obj)
	if err != nil {
		return fmt.Errorf("failed to read base binary: %w", err)
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	bw, err := zw.Create("enx-agent" + ext)
	if err != nil {
		return err
	}
	if _, err := bw.Write(binaryData); err != nil {
		return err
	}

	cw, err := zw.Create("agent.enx")
	if err != nil {
		return err
	}
	if _, err := cw.Write(configENX); err != nil {
		return err
	}

	if err := zw.Close(); err != nil {
		return err
	}

	_, err = w.minioClient.PutObject(ctx, w.bucket, artifactPath,
		bytes.NewReader(buf.Bytes()), int64(buf.Len()),
		minio.PutObjectOptions{ContentType: "application/zip"},
	)
	if err != nil {
		return fmt.Errorf("failed to upload fallback package: %w", err)
	}

	return w.markReady(ctx, p.PackageID, artifactPath)
}

func (w *PackageBuildWorker) markReady(ctx context.Context, packageID, artifactPath string) error {
	result := w.db.WithContext(ctx).Table("download_packages").
		Where("id = ?", packageID).
		Updates(map[string]interface{}{
			"status":        "ready",
			"sign_status":   "signed",
			"artifact_path": artifactPath,
			"updated_at":    time.Now(),
		})
	if result.Error != nil {
		return result.Error
	}
	slog.Info("Package built successfully", "package_id", packageID, "artifact_path", artifactPath)
	return nil
}

// buildConfigENX generates agent.enx (TOML format) for the download package.
func (w *PackageBuildWorker) buildConfigENX(p packageBuildPayload) []byte {
	platformURL := os.Getenv("ENX_PLATFORM_API_PUBLIC_BASE_URL")
	if platformURL == "" {
		platformURL = os.Getenv("ENX_PLATFORM_URL")
	}
	if platformURL == "" {
		platformURL = "http://localhost:8080"
	}

	wsURL := os.Getenv("ENX_SESSION_GATEWAY_PUBLIC_WS_URL")
	if wsURL == "" {
		wsURL = os.Getenv("ENX_WS_URL")
	}
	if wsURL == "" {
		wsURL = "ws://localhost:8081"
	}
	if !strings.HasSuffix(wsURL, "/ws/v1/sessions") {
		wsURL = strings.TrimRight(wsURL, "/") + "/ws/v1/sessions"
	}

	var buf bytes.Buffer
	buf.WriteString("# EnvNexus Agent Configuration\n")
	buf.WriteString("# Place this file next to the installer before running it,\n")
	buf.WriteString("# or copy to the agent data directory after installation.\n\n")
	buf.WriteString(fmt.Sprintf("platform_url = %q\n", platformURL))
	buf.WriteString(fmt.Sprintf("ws_url = %q\n", wsURL))
	buf.WriteString(fmt.Sprintf("enrollment_token = %q\n", "auto_generated_token_for_"+p.TenantID))

	if p.ActivationMode != "" {
		buf.WriteString(fmt.Sprintf("activation_mode = %q\n", p.ActivationMode))
	}
	if (p.ActivationMode == "auto" || p.ActivationMode == "both") && p.ActivationKey != "" {
		buf.WriteString(fmt.Sprintf("activation_key = %q\n", p.ActivationKey))
	}

	return buf.Bytes()
}
