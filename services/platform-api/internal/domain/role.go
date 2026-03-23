package domain

import "time"

type Role struct {
	ID              string
	TenantID        string
	Name            string
	PermissionsJSON string
	Status          string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
