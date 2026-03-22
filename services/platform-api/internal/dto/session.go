package dto

import "time"

type SessionResponse struct {
	ID            string     `json:"id"`
	TenantID      string     `json:"tenant_id"`
	DeviceID      string     `json:"device_id"`
	Transport     string     `json:"transport"`
	Status        string     `json:"status"`
	InitiatorType string     `json:"initiator_type"`
	StartedAt     time.Time  `json:"started_at"`
	EndedAt       *time.Time `json:"ended_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}
