package dto

import "time"

type ModelProfileResponse struct {
	ID                     string    `json:"id"`
	TenantID               string    `json:"tenant_id"`
	Name                   string    `json:"name"`
	Provider               string    `json:"provider"`
	BaseURL                string    `json:"base_url"`
	ModelName              string    `json:"model_name"`
	APIKey                 string    `json:"api_key"`
	ParamsJSON             string    `json:"params_json"`
	SecretMode             string    `json:"secret_mode"`
	FallbackModelProfileID *string   `json:"fallback_model_profile_id"`
	Status                 string    `json:"status"`
	Version                int       `json:"version"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

type CreateModelProfileRequest struct {
	Name       string `json:"name" binding:"required"`
	Provider   string `json:"provider" binding:"required"`
	BaseURL    string `json:"base_url" binding:"required"`
	ModelName  string `json:"model_name" binding:"required"`
	APIKey     string `json:"api_key"`
	ParamsJSON string `json:"params_json" binding:"required"`
	SecretMode string `json:"secret_mode" binding:"required"`
}

type UpdateModelProfileRequest struct {
	Name       string `json:"name"`
	Provider   string `json:"provider"`
	BaseURL    string `json:"base_url"`
	ModelName  string `json:"model_name"`
	APIKey     string `json:"api_key"`
	ParamsJSON string `json:"params_json"`
	SecretMode string `json:"secret_mode"`
	Status     string `json:"status"`
}
