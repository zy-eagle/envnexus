package config

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	toml "github.com/pelletier/go-toml/v2"
)

type AgentConfig struct {
	PlatformURL                 string `json:"platform_url" toml:"platform_url"`
	WSURL                       string `json:"ws_url" toml:"ws_url"`
	EnrollmentToken             string `json:"enrollment_token" toml:"enrollment_token"`
	ConfigVersion               int    `json:"config_version" toml:"config_version"`
	HeartbeatSeconds            int    `json:"heartbeat_seconds" toml:"heartbeat_seconds"`
	AgentVersion                string `json:"agent_version" toml:"agent_version"`
	// DistributionPackageVersion is the console download-package semver (installer/ZIP bundle).
	// Used for /agent/v1/check-update — not the embedded agent-core binary version.
	DistributionPackageVersion string `json:"distribution_package_version,omitempty" toml:"distribution_package_version,omitempty"`
	ActivationMode              string `json:"activation_mode,omitempty" toml:"activation_mode,omitempty"`
	ActivationKey               string `json:"activation_key,omitempty" toml:"activation_key,omitempty"`
	AutoUpdate                  bool   `json:"auto_update,omitempty" toml:"auto_update,omitempty"`
}

// CLIOverrides holds values from command-line flags (highest priority).
type CLIOverrides struct {
	PlatformURL    string
	WSURL          string
	ActivationMode string
	ActivationKey  string
	DataDir        string
}

type Manager struct {
	mu        sync.RWMutex
	config    *AgentConfig
	configDir string
}

func NewManager(configDir string) *Manager {
	m := &Manager{
		configDir: configDir,
		config: &AgentConfig{
			PlatformURL:      envOrDefault("ENX_PLATFORM_URL", "http://localhost:8080"),
			WSURL:            envOrDefault("ENX_WS_URL", "ws://localhost:8081/ws/v1/sessions"),
			EnrollmentToken:  os.Getenv("ENX_ENROLLMENT_TOKEN"),
			ConfigVersion:    0,
			HeartbeatSeconds: 30,
			AgentVersion:     "0.3.0",
		},
	}
	m.loadFromBundledConfig()
	m.loadFromExecutable()
	m.loadFromDisk()
	slog.Info("[config] Final config loaded",
		"platform_url", m.config.PlatformURL,
		"has_enrollment_token", m.config.EnrollmentToken != "",
		"config_dir", configDir)
	return m
}

func (m *Manager) Get() *AgentConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	copy := *m.config
	return &copy
}

func (m *Manager) ConfigDir() string {
	return m.configDir
}

// ApplyCLIOverrides applies command-line flag values (highest priority, overrides everything).
func (m *Manager) ApplyCLIOverrides(o CLIOverrides) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if o.PlatformURL != "" {
		m.config.PlatformURL = o.PlatformURL
	}
	if o.WSURL != "" {
		m.config.WSURL = o.WSURL
	}
	if o.ActivationMode != "" {
		m.config.ActivationMode = o.ActivationMode
	}
	if o.ActivationKey != "" {
		m.config.ActivationKey = o.ActivationKey
	}
}

func (m *Manager) Update(fn func(c *AgentConfig)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	fn(m.config)
	m.saveToDisk()
}

// loadFromBundledConfig reads agent.enx (TOML) or config.json from the executable's
// directory and common parent directories (for Electron packaged apps where the exe
// lives in resources/bin/ but agent.enx is in the install root).
func (m *Manager) loadFromBundledConfig() {
	exePath, err := os.Executable()
	if err != nil {
		return
	}
	exeDir := filepath.Dir(exePath)

	searchDirs := []string{
		exeDir,
		filepath.Dir(exeDir),                       // parent (e.g. resources/ -> install root)
		filepath.Join(filepath.Dir(exeDir), ".."),   // grandparent (e.g. resources/bin/ -> install root)
	}

	for _, dir := range searchDirs {
		if m.tryLoadTOML(filepath.Join(dir, "agent.enx")) {
			slog.Info("[config] Loaded bundled agent.enx", "dir", dir)
			return
		}
	}
	for _, dir := range searchDirs {
		if m.tryLoadJSON(filepath.Join(dir, "config.json")) {
			slog.Info("[config] Loaded bundled config.json", "dir", dir)
			return
		}
	}
}

func (m *Manager) loadFromDisk() {
	enxPath := filepath.Join(m.configDir, "agent.enx")
	if m.tryLoadTOML(enxPath) {
		slog.Info("[config] Loaded config from disk", "path", enxPath,
			"has_enrollment_token", m.config.EnrollmentToken != "")
		return
	}
	// Fallback to legacy JSON
	path := filepath.Join(m.configDir, "agent_config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var diskConfig AgentConfig
	if err := json.Unmarshal(data, &diskConfig); err != nil {
		slog.Error("[config] Failed to parse config file", "error", err)
		return
	}
	m.applyPartial(&diskConfig)
	if diskConfig.ConfigVersion > 0 {
		m.config.ConfigVersion = diskConfig.ConfigVersion
	}
	if diskConfig.HeartbeatSeconds > 0 {
		m.config.HeartbeatSeconds = diskConfig.HeartbeatSeconds
	}
}

func (m *Manager) tryLoadTOML(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var cfg AgentConfig
	if err := toml.Unmarshal(data, &cfg); err != nil {
		slog.Warn("[config] Found .enx file but failed to parse TOML", "path", path, "error", err)
		return false
	}
	m.applyPartial(&cfg)
	if cfg.ConfigVersion > 0 {
		m.config.ConfigVersion = cfg.ConfigVersion
	}
	if cfg.HeartbeatSeconds > 0 {
		m.config.HeartbeatSeconds = cfg.HeartbeatSeconds
	}
	return true
}

func (m *Manager) tryLoadJSON(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var cfg AgentConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		slog.Warn("[config] Found JSON config but failed to parse", "path", path, "error", err)
		return false
	}
	m.applyPartial(&cfg)
	return true
}

// loadFromExecutable attempts to read a JSON payload appended to the end of the executable.
func (m *Manager) loadFromExecutable() {
	exePath, err := os.Executable()
	if err != nil {
		return
	}

	fileInfo, err := os.Stat(exePath)
	if err != nil {
		return
	}

	readSize := int64(4096)
	if fileInfo.Size() < readSize {
		readSize = fileInfo.Size()
	}

	f, err := os.Open(exePath)
	if err != nil {
		return
	}
	defer f.Close()

	buf := make([]byte, readSize)
	_, err = f.ReadAt(buf, fileInfo.Size()-readSize)
	if err != nil {
		return
	}

	magic := []byte("ENX_CONF_START:")
	idx := bytes.LastIndex(buf, magic)
	if idx == -1 {
		return
	}

	jsonPayload := buf[idx+len(magic):]
	jsonPayload = bytes.TrimRight(jsonPayload, "\x00\n\r\t ")

	var injectedConfig AgentConfig
	if err := json.Unmarshal(jsonPayload, &injectedConfig); err != nil {
		slog.Error("[config] Found injected config but failed to parse JSON", "error", err)
		return
	}

	slog.Info("[config] Successfully loaded injected config from executable")
	m.applyPartial(&injectedConfig)
}

func (m *Manager) applyPartial(src *AgentConfig) {
	if src.PlatformURL != "" {
		m.config.PlatformURL = src.PlatformURL
	}
	if src.WSURL != "" {
		m.config.WSURL = src.WSURL
	}
	if src.EnrollmentToken != "" {
		m.config.EnrollmentToken = src.EnrollmentToken
	}
	if src.ActivationMode != "" {
		m.config.ActivationMode = src.ActivationMode
	}
	if src.ActivationKey != "" {
		m.config.ActivationKey = src.ActivationKey
	}
	if src.AgentVersion != "" {
		m.config.AgentVersion = src.AgentVersion
	}
	if src.DistributionPackageVersion != "" {
		m.config.DistributionPackageVersion = src.DistributionPackageVersion
	}
	if src.AutoUpdate {
		m.config.AutoUpdate = true
	}
}

// saveToDisk persists config in TOML (.enx) format.
func (m *Manager) saveToDisk() {
	path := filepath.Join(m.configDir, "agent.enx")
	data, err := toml.Marshal(m.config)
	if err != nil {
		slog.Error("[config] Failed to marshal config", "error", err)
		return
	}

	header := []byte("# EnvNexus Agent Configuration\n# Auto-generated — do not edit manually unless you know what you are doing.\n\n")
	_ = os.MkdirAll(m.configDir, 0755)
	if err := os.WriteFile(path, append(header, data...), 0600); err != nil {
		slog.Error("[config] Failed to save config", "error", err)
	}

	// Clean up legacy JSON config if .enx was written successfully
	legacyPath := filepath.Join(m.configDir, "agent_config.json")
	if _, err := os.Stat(legacyPath); err == nil {
		_ = os.Remove(legacyPath)
		slog.Info("[config] Removed legacy agent_config.json (migrated to agent.enx)")
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
