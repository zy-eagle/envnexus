package dto

import "time"

type AuditEventResponse struct {
	ID               string `json:"id"`
	TenantID         string `json:"tenant_id"`
	DeviceID         string `json:"device_id,omitempty"`
	SessionID        string `json:"session_id,omitempty"`
	EventType        string `json:"event_type"`
	EventPayloadJSON string `json:"event_payload_json"`
	CreatedAt        time.Time `json:"created_at"`
}
