package domain

import "time"

type AuditEvent struct {
	ID               string
	TenantID         string
	DeviceID         *string
	SessionID        *string
	EventType        string
	EventPayloadJSON string
	Archived         bool
	CreatedAt        time.Time
}

func (a *AuditEvent) TableName() string { return "audit_events" }
