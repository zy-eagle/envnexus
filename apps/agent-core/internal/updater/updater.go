package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

type Config struct {
	PlatformURL    string
	DeviceToken    string
	CurrentVersion string
	AutoUpdate     bool
	CheckInterval  time.Duration
	DataDir        string
}

type UpdateInfo struct {
	HasUpdate      bool   `json:"has_update"`
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	DownloadURL    string `json:"download_url"`
	PackageID      string `json:"package_id"`
	Checksum       string `json:"checksum"`
	ArtifactSize   int64  `json:"artifact_size"`
	Message        string `json:"message"`
}

type StatusListener func(status Status)

type Status struct {
	State          string `json:"state"`
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version,omitempty"`
	Progress       int    `json:"progress,omitempty"`
	Error          string `json:"error,omitempty"`
	PendingRestart bool   `json:"pending_restart"`
}

type Updater struct {
	config     Config
	httpClient *http.Client

	mu             sync.RWMutex
	latestInfo     *UpdateInfo
	pendingBinary  string
	status         Status
	listeners      []StatusListener
}

func New(cfg Config) *Updater {
	if cfg.CheckInterval == 0 {
		cfg.CheckInterval = 1 * time.Hour
	}
	return &Updater{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		status: Status{
			State:          "idle",
			CurrentVersion: cfg.CurrentVersion,
		},
	}
}

func (u *Updater) OnStatus(fn StatusListener) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.listeners = append(u.listeners, fn)
}

func (u *Updater) GetStatus() Status {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.status
}

func (u *Updater) setStatus(s Status) {
	u.mu.Lock()
	u.status = s
	listeners := make([]StatusListener, len(u.listeners))
	copy(listeners, u.listeners)
	u.mu.Unlock()

	for _, fn := range listeners {
		fn(s)
	}
}

func (u *Updater) CheckForUpdate(ctx context.Context) (*UpdateInfo, error) {
	url := fmt.Sprintf("%s/agent/v1/check-update?current_version=%s&platform=%s&arch=%s",
		u.config.PlatformURL, u.config.CurrentVersion, runtime.GOOS, runtime.GOARCH)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+u.config.DeviceToken)

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("check-update request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("check-update returned status %d", resp.StatusCode)
	}

	var apiResp struct {
		Data UpdateInfo `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	u.mu.Lock()
	u.latestInfo = &apiResp.Data
	u.mu.Unlock()

	return &apiResp.Data, nil
}

// DownloadUpdate downloads the new binary to a staging area.
// Returns the path to the downloaded file.
func (u *Updater) DownloadUpdate(ctx context.Context, info *UpdateInfo) (string, error) {
	if info.DownloadURL == "" {
		return "", fmt.Errorf("no download URL available")
	}

	u.setStatus(Status{
		State:          "downloading",
		CurrentVersion: u.config.CurrentVersion,
		LatestVersion:  info.LatestVersion,
		Progress:       0,
	})

	stageDir := filepath.Join(u.config.DataDir, "updates")
	if err := os.MkdirAll(stageDir, 0755); err != nil {
		return "", fmt.Errorf("create staging dir: %w", err)
	}

	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	stagePath := filepath.Join(stageDir, fmt.Sprintf("enx-agent-%s%s", info.LatestVersion, ext))

	// Skip if already downloaded and checksum matches
	if info.Checksum != "" {
		if existing, err := os.Open(stagePath); err == nil {
			h := sha256.New()
			io.Copy(h, existing)
			existing.Close()
			if hex.EncodeToString(h.Sum(nil)) == info.Checksum {
				slog.Info("[updater] Binary already staged and verified", "path", stagePath)
				u.mu.Lock()
				u.pendingBinary = stagePath
				u.mu.Unlock()
				u.setStatus(Status{
					State:          "ready",
					CurrentVersion: u.config.CurrentVersion,
					LatestVersion:  info.LatestVersion,
					PendingRestart: true,
				})
				return stagePath, nil
			}
		}
	}

	dlClient := &http.Client{Timeout: 10 * time.Minute}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, info.DownloadURL, nil)
	if err != nil {
		return "", fmt.Errorf("create download request: %w", err)
	}

	resp, err := dlClient.Do(req)
	if err != nil {
		u.setStatus(Status{State: "error", CurrentVersion: u.config.CurrentVersion, Error: err.Error()})
		return "", fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		u.setStatus(Status{State: "error", CurrentVersion: u.config.CurrentVersion, Error: fmt.Sprintf("HTTP %d", resp.StatusCode)})
		return "", fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp(stageDir, "enx-agent-dl-*.tmp")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	hasher := sha256.New()
	writer := io.MultiWriter(tmpFile, hasher)

	totalSize := resp.ContentLength
	var written int64
	buf := make([]byte, 64*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, wErr := writer.Write(buf[:n]); wErr != nil {
				tmpFile.Close()
				os.Remove(tmpPath)
				return "", fmt.Errorf("write: %w", wErr)
			}
			written += int64(n)
			if totalSize > 0 {
				progress := int(float64(written) / float64(totalSize) * 100)
				u.setStatus(Status{
					State:          "downloading",
					CurrentVersion: u.config.CurrentVersion,
					LatestVersion:  info.LatestVersion,
					Progress:       progress,
				})
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			tmpFile.Close()
			os.Remove(tmpPath)
			return "", fmt.Errorf("read: %w", readErr)
		}
	}
	tmpFile.Close()

	// Verify checksum if provided
	if info.Checksum != "" {
		computed := hex.EncodeToString(hasher.Sum(nil))
		if computed != info.Checksum {
			os.Remove(tmpPath)
			return "", fmt.Errorf("checksum mismatch: expected %s, got %s", info.Checksum, computed)
		}
		slog.Info("[updater] Checksum verified", "sha256", computed)
	}

	// Make executable and move to final staging path
	if err := os.Chmod(tmpPath, 0755); err != nil {
		slog.Warn("[updater] chmod failed", "error", err)
	}
	if err := os.Rename(tmpPath, stagePath); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("rename staged binary: %w", err)
	}

	u.mu.Lock()
	u.pendingBinary = stagePath
	u.mu.Unlock()

	u.setStatus(Status{
		State:          "ready",
		CurrentVersion: u.config.CurrentVersion,
		LatestVersion:  info.LatestVersion,
		PendingRestart: true,
	})

	slog.Info("[updater] Update downloaded and staged", "version", info.LatestVersion, "path", stagePath)
	return stagePath, nil
}

// ApplyUpdate replaces the current binary with the staged one.
// On Windows, the old binary is renamed (since the running process holds a lock)
// and the new one is placed at the original path.
// The caller should restart the process after this returns.
func (u *Updater) ApplyUpdate() error {
	u.mu.RLock()
	staged := u.pendingBinary
	u.mu.RUnlock()

	if staged == "" {
		return fmt.Errorf("no pending update")
	}

	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve current executable: %w", err)
	}
	currentExe, err = filepath.EvalSymlinks(currentExe)
	if err != nil {
		return fmt.Errorf("resolve symlinks: %w", err)
	}

	backupPath := currentExe + ".bak"

	// Remove previous backup if exists
	os.Remove(backupPath)

	// Rename current -> backup
	if err := os.Rename(currentExe, backupPath); err != nil {
		return fmt.Errorf("backup current binary: %w", err)
	}

	// Move staged -> current
	if err := os.Rename(staged, currentExe); err != nil {
		// Rollback: restore backup
		os.Rename(backupPath, currentExe)
		return fmt.Errorf("install new binary: %w", err)
	}

	if err := os.Chmod(currentExe, 0755); err != nil {
		slog.Warn("[updater] chmod on new binary failed", "error", err)
	}

	u.mu.Lock()
	u.pendingBinary = ""
	u.mu.Unlock()

	slog.Info("[updater] Binary replaced successfully, restart required", "old_backup", backupPath)

	u.setStatus(Status{
		State:          "applied",
		CurrentVersion: u.config.CurrentVersion,
		PendingRestart: true,
	})

	return nil
}

// Run starts the periodic update check loop. Blocks until context is cancelled.
func (u *Updater) Run(ctx context.Context) {
	// Initial check after a short delay
	select {
	case <-ctx.Done():
		return
	case <-time.After(30 * time.Second):
	}

	u.checkAndMaybeDownload(ctx)

	ticker := time.NewTicker(u.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			u.checkAndMaybeDownload(ctx)
		}
	}
}

func (u *Updater) checkAndMaybeDownload(ctx context.Context) {
	u.setStatus(Status{
		State:          "checking",
		CurrentVersion: u.config.CurrentVersion,
	})

	info, err := u.CheckForUpdate(ctx)
	if err != nil {
		slog.Warn("[updater] Check failed", "error", err)
		u.setStatus(Status{
			State:          "idle",
			CurrentVersion: u.config.CurrentVersion,
			Error:          err.Error(),
		})
		return
	}

	if !info.HasUpdate {
		slog.Debug("[updater] No update available")
		u.setStatus(Status{
			State:          "idle",
			CurrentVersion: u.config.CurrentVersion,
		})
		return
	}

	slog.Info("[updater] Update available", "current", u.config.CurrentVersion, "latest", info.LatestVersion)

	if u.config.AutoUpdate {
		if _, err := u.DownloadUpdate(ctx, info); err != nil {
			slog.Error("[updater] Download failed", "error", err)
			u.setStatus(Status{
				State:          "error",
				CurrentVersion: u.config.CurrentVersion,
				LatestVersion:  info.LatestVersion,
				Error:          err.Error(),
			})
		}
	} else {
		u.setStatus(Status{
			State:          "update_available",
			CurrentVersion: u.config.CurrentVersion,
			LatestVersion:  info.LatestVersion,
		})
	}
}
