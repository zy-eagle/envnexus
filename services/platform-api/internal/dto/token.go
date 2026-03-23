package dto

import "time"

type CreateTokenRequest struct {
	AgentProfileID string `json:"agent_profile_id"`
	MaxUses        int    `json:"max_uses" binding:"required,min=1"`
	ExpiresIn      int    `json:"expires_in" binding:"required,min=1"`
}

type TokenResponse struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Token     string    `json:"token"`
	MaxUses   int       `json:"max_uses"`
	UsedCount int       `json:"used_count"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

type AgentEnrollResponse struct {
	DeviceID      string `json:"device_id"`
	TenantID      string `json:"tenant_id"`
	DeviceToken   string `json:"device_token"`
	ConfigVersion int    `json:"config_version"`
}
