package device

import (
	"crypto/rand"
	"encoding/hex"
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
	return &IdentityManager{
		configDir: configDir,
	}
}

// GetOrCreateDeviceID retrieves the existing device ID or generates a new one.
func (m *IdentityManager) GetOrCreateDeviceID() (string, error) {
	idFile := filepath.Join(m.configDir, "device_id")

	// Try to read existing ID
	data, err := os.ReadFile(idFile)
	if err == nil && len(data) > 0 {
		return string(data), nil
	}

	// Generate new ID
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate device ID: %w", err)
	}
	newID := hex.EncodeToString(bytes)

	// Save new ID
	if err := os.MkdirAll(m.configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config dir: %w", err)
	}
	if err := os.WriteFile(idFile, []byte(newID), 0600); err != nil {
		return "", fmt.Errorf("failed to write device ID: %w", err)
	}

	return newID, nil
}

// SaveIdentity saves the full identity including tenant and token
func (m *IdentityManager) SaveIdentity(identity *Identity) error {
	if err := os.MkdirAll(m.configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	idFile := filepath.Join(m.configDir, "identity.json")
	
	// We'll use a simple JSON format for now
	data := fmt.Sprintf(`{"device_id":"%s","tenant_id":"%s","token":"%s"}`, 
		identity.DeviceID, identity.TenantID, identity.Token)
		
	if err := os.WriteFile(idFile, []byte(data), 0600); err != nil {
		return fmt.Errorf("failed to write identity: %w", err)
	}
	return nil
}

// GetIdentity retrieves the saved identity
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
