package domain

import "time"

type ModelProfile struct {
	ID                     string
	TenantID               string
	Name                   string
	Provider               string
	BaseURL                string
	ModelName              string
	APIKey                 string `gorm:"column:api_key"`
	ParamsJSON             string // Stored as JSON string
	SecretMode             string
	FallbackModelProfileID *string
	Status                 string
	Version                int
	CreatedAt              time.Time
	UpdatedAt              time.Time
	DeletedAt              *time.Time
}
