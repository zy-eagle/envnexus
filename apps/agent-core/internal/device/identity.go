package device

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Identity struct {
	DeviceID string `json:"device_id"`
	TenantID string `json:"tenant_id"`
	Token    string `json:"token"`
}

type IdentityManager struct {
	configDir string
}

func NewIdentityManager(configDir string) *IdentityManager {
	return &IdentityManager{configDir: configDir}
}

func (m *IdentityManager) GetOrCreateDeviceID() (string, error) {
	idFile := filepath.Join(m.configDir, "device_id")

	data, err := os.ReadFile(idFile)
	if err == nil && len(data) > 0 {
		return string(data), nil
	}

	return "", fmt.Errorf("no device identity found")
}

func (m *IdentityManager) SaveIdentity(identity *Identity) error {
	if err := os.MkdirAll(m.configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	if err := os.WriteFile(filepath.Join(m.configDir, "device_id"), []byte(identity.DeviceID), 0600); err != nil {
		return fmt.Errorf("failed to write device_id: %w", err)
	}

	idFile := filepath.Join(m.configDir, "identity.json")
	data, err := json.MarshalIndent(identity, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal identity: %w", err)
	}
	if err := os.WriteFile(idFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write identity: %w", err)
	}
	return nil
}

func (m *IdentityManager) GetIdentity() (*Identity, error) {
	idFile := filepath.Join(m.configDir, "identity.json")
	data, err := os.ReadFile(idFile)
	if err != nil {
		return nil, err
	}

	var id Identity
	if err := json.Unmarshal(data, &id); err != nil {
		return nil, err
	}
	return &id, nil
}

func (m *IdentityManager) HasIdentity() bool {
	id, err := m.GetIdentity()
	return err == nil && id != nil && id.Token != ""
}
