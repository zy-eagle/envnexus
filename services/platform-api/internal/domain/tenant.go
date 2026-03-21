package domain

import "time"

type TenantStatus string

const (
	TenantStatusDraft     TenantStatus = "draft"
	TenantStatusActive    TenantStatus = "active"
	TenantStatusSuspended TenantStatus = "suspended"
	TenantStatusArchived  TenantStatus = "archived"
)

// Tenant represents a customer workspace in the platform.
type Tenant struct {
	ID        string
	Name      string
	Slug      string
	Status    TenantStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewTenant creates a new tenant with default draft status.
func NewTenant(id, name, slug string) *Tenant {
	return &Tenant{
		ID:        id,
		Name:      name,
		Slug:      slug,
		Status:    TenantStatusDraft,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// Activate transitions the tenant to active state.
func (t *Tenant) Activate() {
	t.Status = TenantStatusActive
	t.UpdatedAt = time.Now()
}
