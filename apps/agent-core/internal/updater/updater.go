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
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// execCommandContext wraps exec.CommandContext — extracted for testability.
var execCommandContext = exec.CommandContext

// CoreInstallPathFile is stored in DataDir so the desktop shell can locate the
// binary that os.Executable reported even when local HTTP or JSON fails.
const CoreInstallPathFile = "core_install_path.json"

type coreInstallPathRecord struct {
	Path         string `json:"path"`
	UpdatedUnix  int64  `json:"updated_unix"`
}

func persistCoreInstallPointer(dataDir, abs string) {
	if dataDir == "" || abs == "" {
		return
	}
	rec := coreInstallPathRecord{
		Path:        filepath.Clean(abs),
		UpdatedUnix: time.Now().Unix(),
	}
	b, err := json.Marshal(rec)
	if err != nil {
		slog.Warn("[updater] marshal core install pointer", "error", err)
		return
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		slog.Warn("[updater] mkdir for core install pointer", "error", err)
		return
	}
	target := filepath.Join(dataDir, CoreInstallPathFile)
	tmp := target + ".tmp"
	if err := os.WriteFile(tmp, b, 0644); err != nil {
		slog.Warn("[updater] write core install pointer tmp", "error", err)
		return
	}
	if err := os.Rename(tmp, target); err != nil {
		slog.Warn("[updater] rename core install pointer", "error", err)
		_ = os.Remove(tmp)
		return
	}
	slog.Info("[updater] Wrote core install pointer for desktop shell", "file", target, "path", rec.Path)
}

// ConfigUpdater is called after a successful apply to persist the new
// distribution_package_version so subsequent check-update requests use it.
type ConfigUpdater func(newVersion string)

type Config struct {
	PlatformURL    string
	DeviceToken    string
	// CurrentVersion is the semver sent to /agent/v1/check-update (distribution bundle by default).
	CurrentVersion string
	// CoreVersion is the embedded agent-core binary semver (for status/diagnostics only).
	CoreVersion    string
	AutoUpdate     bool
	CheckInterval  time.Duration
	DataDir        string
	// OnVersionApplied is called after ApplyUpdate succeeds to persist the new
	// distribution version into config.  May be nil.
	OnVersionApplied ConfigUpdater
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
	CoreVersion    string `json:"core_version,omitempty"`
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
			CoreVersion:    cfg.CoreVersion,
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
	if s.CoreVersion == "" {
		s.CoreVersion = u.config.CoreVersion
	}
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

	slog.Info("[updater] check-update request",
		"platform_url", u.config.PlatformURL,
		"current_version", u.config.CurrentVersion,
		"platform", runtime.GOOS,
		"arch", runtime.GOARCH,
		"url", url,
	)

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
// On Windows the old binary is renamed first (the running process holds a
// handle) and the new one is copied (not renamed) into place so cross-volume
// scenarios work reliably. After the copy we open the new file for reading to
// trigger the real-time AV scan *before* the desktop shell tries to spawn it,
// dramatically reducing "spawn UNKNOWN" failures.
// The returned path is the absolute filesystem path where the new binary was
// installed (same slot as os.Executable); the desktop shell should spawn this
// exact path to avoid heuristic mismatches on Windows.
func (u *Updater) ApplyUpdate() (installedPath string, err error) {
	u.mu.RLock()
	staged := u.pendingBinary
	u.mu.RUnlock()

	if staged == "" {
		return "", fmt.Errorf("no pending update")
	}

	currentExe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve current executable: %w", err)
	}
	currentExe, err = filepath.EvalSymlinks(currentExe)
	if err != nil {
		return "", fmt.Errorf("resolve symlinks: %w", err)
	}
	currentExe = filepath.Clean(currentExe)

	backupPath := currentExe + ".bak"

	os.Remove(backupPath)

	if err := os.Rename(currentExe, backupPath); err != nil {
		return "", fmt.Errorf("backup current binary: %w", err)
	}

	if err := copyFile(staged, currentExe); err != nil {
		os.Rename(backupPath, currentExe)
		return "", fmt.Errorf("install new binary: %w", err)
	}

	if err := os.Chmod(currentExe, 0755); err != nil {
		slog.Warn("[updater] chmod on new binary failed", "error", err)
	}

	// Trigger Windows Defender / AV real-time scan by reading the full file
	// before we return success to the desktop shell.  This way the scan
	// completes (or at least starts) while we still hold control, instead of
	// racing with the first spawn attempt.
	warmupAVScan(currentExe)

	os.Remove(staged)

	u.mu.Lock()
	u.pendingBinary = ""
	u.mu.Unlock()

	slog.Info("[updater] Binary replaced successfully, restart required", "old_backup", backupPath, "installed_path", currentExe)

	persistCoreInstallPointer(u.config.DataDir, currentExe)

	// Persist the new distribution version so that after restart the
	// check-update request carries the updated semver instead of the old one.
	u.mu.RLock()
	newVer := ""
	if u.latestInfo != nil {
		newVer = u.latestInfo.LatestVersion
	}
	u.mu.RUnlock()

	if newVer != "" {
		u.config.CurrentVersion = newVer
		if u.config.OnVersionApplied != nil {
			u.config.OnVersionApplied(newVer)
		}
		slog.Info("[updater] Updated distribution_package_version after apply", "new_version", newVer)
	}

	u.setStatus(Status{
		State:            "applied",
		CurrentVersion:   u.config.CurrentVersion,
		PendingRestart:   true,
	})

	return currentExe, nil
}

// copyFile copies src to dst using a temp file + rename for atomicity within
// the same directory.  Unlike os.Rename this works across volumes/mount points.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer in.Close()

	dir := filepath.Dir(dst)
	tmp, err := os.CreateTemp(dir, ".enx-update-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := io.Copy(tmp, in); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("copy: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("sync: %w", err)
	}
	tmp.Close()

	if err := os.Rename(tmpName, dst); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("rename temp to target: %w", err)
	}
	return nil
}

// warmupAVScan triggers both file-level and process-level AV scans on the
// new binary *before* the Electron shell tries to spawn it as a long-running
// process.  This dramatically reduces "spawn UNKNOWN" failures on Windows.
//
// Phase 1 — file-level: read the entire file to trigger content-based scan,
//   then poll os.Open until the AV releases its read lock.
// Phase 2 — process-level: execute the binary with the harmless "version"
//   subcommand.  This fires NtCreateUserProcess which triggers Defender's
//   AMSI / PPL checks.  After this returns the exe is "known" and subsequent
//   CreateProcess calls succeed without delay.
func warmupAVScan(exePath string) {
	if runtime.GOOS != "windows" {
		return
	}

	// Phase 1: file read
	f, err := os.Open(exePath)
	if err != nil {
		slog.Warn("[updater] warmup: initial open failed", "error", err)
		return
	}
	io.Copy(io.Discard, f)
	f.Close()

	for i := 0; i < 16; i++ {
		fh, err := os.Open(exePath)
		if err == nil {
			fh.Close()
			slog.Info("[updater] warmup: file ready after content scan", "retries", i)
			break
		}
		if i == 15 {
			slog.Warn("[updater] warmup: file still locked after content scan retries")
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Phase 2: process execution — triggers AMSI / Defender process-creation scan.
	// "version" prints one line and exits 0; any failure here is non-fatal.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := execCommandContext(ctx, exePath, "version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		slog.Warn("[updater] warmup: version probe failed", "error", err, "output", string(out))
	} else {
		slog.Info("[updater] warmup: process-level probe succeeded", "output", strings.TrimSpace(string(out)))
	}
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
