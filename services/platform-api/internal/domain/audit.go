package domain

import (
	"time"
)

type AuditEvent struct {
	ID           string
	TenantID     string
	DeviceID     string
	SessionID    string
	ActionType   string
	Status       string
	Payload      []byte // JSON
	ErrorMessage string
	CreatedAt    time.Time
}
