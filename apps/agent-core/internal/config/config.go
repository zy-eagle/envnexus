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
	m.loadFromDisk()
	m.loadFromExecutable()
	return m
}

func (m *Manager) Get() *AgentConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	copy := *m.config
	return &copy
}

func (m *Manager) Update(fn func(c *AgentConfig)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	fn(m.config)
	m.saveToDisk()
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
	if diskConfig.PlatformURL != "" {
		m.config.PlatformURL = diskConfig.PlatformURL
	}
	if diskConfig.WSURL != "" {
		m.config.WSURL = diskConfig.WSURL
	}
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
	if injectedConfig.PlatformURL != "" {
		m.config.PlatformURL = injectedConfig.PlatformURL
	}
	if injectedConfig.EnrollmentToken != "" {
		m.config.EnrollmentToken = injectedConfig.EnrollmentToken
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
