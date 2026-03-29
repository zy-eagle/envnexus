package domain

import "time"

type ModelProfile struct {
	ID                     string     `json:"id"`
	TenantID               string     `json:"tenant_id"`
	Name                   string     `json:"name"`
	Provider               string     `json:"provider"`
	BaseURL                string     `json:"base_url"`
	ModelName              string     `json:"model_name"`
	APIKey                 string     `json:"api_key" gorm:"column:api_key"`
	ParamsJSON             string     `json:"params_json"`
	SecretMode             string     `json:"secret_mode"`
	FallbackModelProfileID *string    `json:"fallback_model_profile_id"`
	Status                 string     `json:"status"`
	Version                int        `json:"version"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
	DeletedAt              *time.Time `json:"deleted_at,omitempty"`
}
