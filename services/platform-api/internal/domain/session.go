package domain

import "time"

type Session struct {
	ID            string
	TenantID      string
	DeviceID      string
	Transport     string
	Status        string
	InitiatorType string
	StartedAt     time.Time
	EndedAt       *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
