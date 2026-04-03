package dto

import (
	"encoding/json"
	"time"
)

type DeviceResponse struct {
	ID               string          `json:"id"`
	TenantID         string          `json:"tenant_id"`
	AgentProfileID   string          `json:"agent_profile_id"`
	DeviceName       string          `json:"device_name"`
	Hostname         string          `json:"hostname,omitempty"`
	Platform         string          `json:"platform"`
	Arch             string          `json:"arch"`
	RuntimeMetadata  json.RawMessage `json:"runtime_metadata,omitempty"`
	EnvironmentType  string          `json:"environment_type"`
	AgentVersion    string     `json:"agent_version"`
	Status          string     `json:"status"`
	PolicyVersion   int        `json:"policy_version"`
	LastSeenAt      *time.Time `json:"last_seen_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type UpdateDeviceRequest struct {
	DeviceName string `json:"device_name"`
	Status     string `json:"status"`
}
