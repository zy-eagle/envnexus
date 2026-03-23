package domain

import "time"

type PolicyProfile struct {
	ID         string
	TenantID   string
	Name       string
	PolicyJSON string
	Status     string
	Version    int
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  *time.Time
}

func (p *PolicyProfile) TableName() string { return "policy_profiles" }
