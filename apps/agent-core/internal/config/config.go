package config

import (
	"encoding/json"
	"log"
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
			WSURL:            envOrDefault("ENX_WS_URL", "ws://localhost:8081/ws/v1/agent"),
			EnrollmentToken:  os.Getenv("ENX_ENROLLMENT_TOKEN"),
			ConfigVersion:    0,
			HeartbeatSeconds: 30,
			AgentVersion:     "0.1.0",
		},
	}
	m.loadFromDisk()
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
		log.Printf("[config] Failed to parse config file: %v\n", err)
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

func (m *Manager) saveToDisk() {
	path := filepath.Join(m.configDir, "agent_config.json")
	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		log.Printf("[config] Failed to marshal config: %v\n", err)
		return
	}
	_ = os.MkdirAll(m.configDir, 0755)
	if err := os.WriteFile(path, data, 0600); err != nil {
		log.Printf("[config] Failed to save config: %v\n", err)
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
