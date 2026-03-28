package config

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

type AgentConfig struct {
	PlatformURL      string `json:"platform_url"`
	WSURL            string `json:"ws_url"`
	EnrollmentToken  string `json:"enrollment_token"`
	ConfigVersion    int    `json:"config_version"`
	HeartbeatSeconds int    `json:"heartbeat_seconds"`
	AgentVersion     string `json:"agent_version"`
	ActivationMode   string `json:"activation_mode,omitempty"`
	ActivationKey    string `json:"activation_key,omitempty"`
}

// CLIOverrides holds values from command-line flags (highest priority).
type CLIOverrides struct {
	PlatformURL    string
	WSURL          string
	ActivationMode string
	ActivationKey  string
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
			AgentVersion:     "0.1.0",
		},
	}
	m.loadFromBundledConfig()
	m.loadFromExecutable()
	m.loadFromDisk()
	return m
}

func (m *Manager) Get() *AgentConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	copy := *m.config
	return &copy
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

// loadFromBundledConfig reads config.json from the same directory as the executable.
// This is the config injected by the download package build system (ZIP bundle).
func (m *Manager) loadFromBundledConfig() {
	exePath, err := os.Executable()
	if err != nil {
		return
	}
	configPath := filepath.Join(filepath.Dir(exePath), "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return
	}
	var bundled AgentConfig
	if err := json.Unmarshal(data, &bundled); err != nil {
		slog.Warn("[config] Found bundled config.json but failed to parse", "error", err)
		return
	}
	slog.Info("[config] Loaded bundled config.json from package directory")
	m.applyPartial(&bundled)
}

func (m *Manager) loadFromDisk() {
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

// loadFromExecutable attempts to read a JSON payload appended to the end of the executable.
// It looks for a magic string "ENX_CONF_START:" followed by JSON data.
func (m *Manager) loadFromExecutable() {
	exePath, err := os.Executable()
	if err != nil {
		return
	}

	fileInfo, err := os.Stat(exePath)
	if err != nil {
		return
	}

	// Only read the last 4KB to look for the payload
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
		return // No injected config found
	}

	jsonPayload := buf[idx+len(magic):]
	// Trim trailing null bytes or newlines that might have been added
	jsonPayload = bytes.TrimRight(jsonPayload, "\x00\n\r\t ")

	var injectedConfig AgentConfig
	if err := json.Unmarshal(jsonPayload, &injectedConfig); err != nil {
		slog.Error("[config] Found injected config but failed to parse JSON", "error", err)
		return
	}

	slog.Info("[config] Successfully loaded injected config from executable")
	m.applyPartial(&injectedConfig)
}

// applyPartial merges non-empty fields from src into the current config.
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
}

func (m *Manager) saveToDisk() {
	path := filepath.Join(m.configDir, "agent_config.json")
	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		slog.Error("[config] Failed to marshal config", "error", err)
		return
	}
	_ = os.MkdirAll(m.configDir, 0755)
	if err := os.WriteFile(path, data, 0600); err != nil {
		slog.Error("[config] Failed to save config", "error", err)
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
