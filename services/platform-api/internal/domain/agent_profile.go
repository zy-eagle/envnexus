package domain

import "time"

type AgentProfile struct {
	ID               string
	TenantID         string
	Name             string
	ModelProfileID   string
	PolicyProfileID  string
	CapabilitiesJSON string
	UpdateChannel    string
	Status           string
	Version          int
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
