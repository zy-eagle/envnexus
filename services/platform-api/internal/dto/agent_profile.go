package dto

import "time"

type AgentProfileResponse struct {
	ID               string    `json:"id"`
	TenantID         string    `json:"tenant_id"`
	Name             string    `json:"name"`
	ModelProfileID   string    `json:"model_profile_id"`
	PolicyProfileID  string    `json:"policy_profile_id"`
	CapabilitiesJSON string    `json:"capabilities_json"`
	UpdateChannel    string    `json:"update_channel"`
	Status           string    `json:"status"`
	Version          int       `json:"version"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type CreateAgentProfileRequest struct {
	Name             string `json:"name" binding:"required"`
	ModelProfileID   string `json:"model_profile_id" binding:"required"`
	PolicyProfileID  string `json:"policy_profile_id" binding:"required"`
	CapabilitiesJSON string `json:"capabilities_json" binding:"required"`
	UpdateChannel    string `json:"update_channel" binding:"required"`
}

type UpdateAgentProfileRequest struct {
	Name             string `json:"name"`
	ModelProfileID   string `json:"model_profile_id"`
	PolicyProfileID  string `json:"policy_profile_id"`
	CapabilitiesJSON string `json:"capabilities_json"`
	UpdateChannel    string `json:"update_channel"`
	Status           string `json:"status"`
}
