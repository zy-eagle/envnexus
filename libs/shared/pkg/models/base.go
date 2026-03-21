package models

import (
	"time"
)

// BaseEntity contains common fields for DB models
type BaseEntity struct {
	ID        string     `json:"id" gorm:"primaryKey;type:varchar(32)"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" gorm:"index"`
}

// Tenant represents a customer tenant
type Tenant struct {
	BaseEntity
	Name   string `json:"name" gorm:"type:varchar(100);not null"`
	Slug   string `json:"slug" gorm:"type:varchar(50);uniqueIndex;not null"`
	Status string `json:"status" gorm:"type:varchar(20);not null;default:'draft'"` // draft, active, suspended, archived
}

// Device represents an enrolled agent device
type Device struct {
	BaseEntity
	TenantID        string    `json:"tenant_id" gorm:"type:varchar(32);index;not null"`
	Hostname        string    `json:"hostname" gorm:"type:varchar(255)"`
	Platform        string    `json:"platform" gorm:"type:varchar(50)"`
	AgentVersion    string    `json:"agent_version" gorm:"type:varchar(50)"`
	Status          string    `json:"status" gorm:"type:varchar(20);not null;default:'online'"` // online, offline, revoked, quarantined
	LastHeartbeatAt time.Time `json:"last_heartbeat_at"`
}
