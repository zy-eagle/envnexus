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
		interval:    3 * time.Second,
	}
}

type packageBuildPayload struct {
	PackageID       string `json:"package_id"`
	TenantID        string `json:"tenant_id"`
	Platform        string `json:"platform"`
	Arch            string `json:"arch"`
	ActivationMode  string `json:"activation_mode,omitempty"`
	ActivationKey   string `json:"activation_key,omitempty"`
	EnrollmentToken string `json:"enrollment_token,omitempty"`
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

func (w *PackageBuildWorker) updateBuildProgress(ctx context.Context, packageID, stage string, progress int) {
	w.db.WithContext(ctx).Table("download_packages").
		Where("id = ?", packageID).
		Updates(map[string]interface{}{
			"build_stage":    stage,
			"build_progress": progress,
			"updated_at":     time.Now(),
		})
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
		Updates(map[string]interface{}{
			"status":         "building",
			"build_stage":    "initializing",
			"build_progress": 5,
			"updated_at":     time.Now(),
		})

	artifactPath := fmt.Sprintf("packages/%s/%s/EnvNexus-Agent-%s-%s.zip", p.TenantID, p.PackageID, p.Platform, p.Arch)

	if w.minioClient == nil || w.bucket == "" {
		return fmt.Errorf("MinIO not configured")
	}

	w.updateBuildProgress(ctx, p.PackageID, "config", 15)
	configENX := w.buildConfigENX(p)

	w.updateBuildProgress(ctx, p.PackageID, "downloading", 25)
	installerData, installerName, err := w.downloadInstaller(ctx, p.Platform, p.Arch)
	if err != nil {
		if p.Platform != "linux" {
			return fmt.Errorf("desktop installer not found for %s/%s — please run agent-builder first to produce the installer: %w", p.Platform, p.Arch, err)
		}
		slog.Warn("Installer not found for Linux, falling back to raw binary", "error", err)
		return w.buildFallbackPackage(ctx, p, configENX, artifactPath)
	}

	w.updateBuildProgress(ctx, p.PackageID, "packaging", 55)

	folderPrefix := fmt.Sprintf("EnvNexus-Agent-%s-%s/", p.Platform, p.Arch)

	pr, pw := io.Pipe()
	uploadErrCh := make(chan error, 1)

	go func() {
		_, err := w.minioClient.PutObject(ctx, w.bucket, artifactPath,
			pr, -1, // Use -1 to let MinIO handle unknown size (since zip headers add unpredictable overhead)
			minio.PutObjectOptions{ContentType: "application/zip"},
		)
		uploadErrCh <- err
	}()

	zw := zip.NewWriter(pw)
	zipErr := createStoreEntry(zw, folderPrefix+installerName, installerData)
	if zipErr == nil {
		zipErr = createStoreEntry(zw, folderPrefix+"agent.enx", configENX)
	}
	if zipErr == nil {
		zipErr = zw.Close()
	}
	pw.CloseWithError(zipErr)

	w.updateBuildProgress(ctx, p.PackageID, "uploading", 85)

	if uploadErr := <-uploadErrCh; uploadErr != nil {
		if zipErr != nil {
			return fmt.Errorf("zip creation failed: %v; upload also failed: %w", zipErr, uploadErr)
		}
		return fmt.Errorf("failed to upload package: %w", uploadErr)
	}
	if zipErr != nil {
		return fmt.Errorf("failed to create installer bundle: %w", zipErr)
	}

	w.updateBuildProgress(ctx, p.PackageID, "done", 100)
	return w.markReady(ctx, p.PackageID, artifactPath)
}

// downloadInstaller tries to fetch the desktop installer from MinIO.
// Uses StatObject first for fast existence check, then downloads only the matching file.
func (w *PackageBuildWorker) downloadInstaller(ctx context.Context, platform, arch string) ([]byte, string, error) {
	candidates := installerCandidates(platform, arch)

	var foundKey string
	var foundSize int64
	for _, key := range candidates {
		info, err := w.minioClient.StatObject(ctx, w.bucket, "base-packages/"+key, minio.StatObjectOptions{})
		if err != nil {
			continue
		}
		foundKey = key
		foundSize = info.Size
		break
	}
	if foundKey == "" {
		return nil, "", fmt.Errorf("no installer found for %s-%s, tried: %v", platform, arch, candidates)
	}

	obj, err := w.minioClient.GetObject(ctx, w.bucket, "base-packages/"+foundKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, "", fmt.Errorf("failed to get installer %s: %w", foundKey, err)
	}
	defer obj.Close()

	data := make([]byte, 0, foundSize)
	buf := bytes.NewBuffer(data)
	if _, err := io.Copy(buf, obj); err != nil {
		return nil, "", fmt.Errorf("failed to read installer %s: %w", foundKey, err)
	}

	slog.Info("Found installer", "key", foundKey, "size", buf.Len())
	return buf.Bytes(), foundKey, nil
}

func installerCandidates(platform, arch string) []string {
	switch platform {
	case "windows":
		return []string{
			fmt.Sprintf("EnvNexus Agent Setup 0.2.0.exe"), // The exact name uploaded by host build
			fmt.Sprintf("EnvNexus-Agent-Setup-windows-%s.exe", arch),
			fmt.Sprintf("EnvNexus-Agent-Setup-%s.exe", arch),
			"EnvNexus-Agent-Setup-windows-amd64.exe",
		}
	case "linux":
		return []string{
			fmt.Sprintf("EnvNexus Agent-0.2.0.AppImage"), // The exact name uploaded by host build
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

// createStoreEntry adds a file to the ZIP archive using Store method (no compression).
// Installers (.exe, .dmg, .AppImage) are already compressed; re-compressing with
// Deflate wastes CPU and time with negligible size reduction.
func createStoreEntry(w *zip.Writer, name string, data []byte) error {
	header := &zip.FileHeader{
		Name:   name,
		Method: zip.Store,
	}
	header.Modified = time.Now()
	fw, err := w.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = fw.Write(data)
	return err
}

// buildFallbackPackage creates a ZIP with raw binary + agent.enx when no installer is available.
func (w *PackageBuildWorker) buildFallbackPackage(ctx context.Context, p packageBuildPayload, configENX []byte, artifactPath string) error {
	w.updateBuildProgress(ctx, p.PackageID, "downloading", 30)

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

	w.updateBuildProgress(ctx, p.PackageID, "packaging", 55)

	folderPrefix := fmt.Sprintf("EnvNexus-Agent-%s-%s/", p.Platform, p.Arch)

	pr, pw := io.Pipe()
	uploadErrCh := make(chan error, 1)

	go func() {
		_, err := w.minioClient.PutObject(ctx, w.bucket, artifactPath,
			pr, -1, // Use -1 to let MinIO handle unknown size (since zip headers add unpredictable overhead)
			minio.PutObjectOptions{ContentType: "application/zip"},
		)
		uploadErrCh <- err
	}()

	zw := zip.NewWriter(pw)
	zipErr := createStoreEntry(zw, folderPrefix+"enx-agent"+ext, binaryData)
	if zipErr == nil {
		zipErr = createStoreEntry(zw, folderPrefix+"agent.enx", configENX)
	}
	if zipErr == nil {
		zipErr = zw.Close()
	}
	pw.CloseWithError(zipErr)

	w.updateBuildProgress(ctx, p.PackageID, "uploading", 85)

	if uploadErr := <-uploadErrCh; uploadErr != nil {
		if zipErr != nil {
			return fmt.Errorf("zip creation failed: %v; upload also failed: %w", zipErr, uploadErr)
		}
		return fmt.Errorf("failed to upload fallback package: %w", uploadErr)
	}
	if zipErr != nil {
		return fmt.Errorf("failed to create fallback zip: %w", zipErr)
	}

	w.updateBuildProgress(ctx, p.PackageID, "done", 100)
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
	if p.EnrollmentToken != "" {
		buf.WriteString(fmt.Sprintf("enrollment_token = %q\n", p.EnrollmentToken))
	}

	if p.ActivationMode != "" {
		buf.WriteString(fmt.Sprintf("activation_mode = %q\n", p.ActivationMode))
	}
	if (p.ActivationMode == "auto" || p.ActivationMode == "both") && p.ActivationKey != "" {
		buf.WriteString(fmt.Sprintf("activation_key = %q\n", p.ActivationKey))
	}

	return buf.Bytes()
}
