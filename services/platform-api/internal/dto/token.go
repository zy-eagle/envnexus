package dto

import "time"

type CreateTokenRequest struct {
	MaxUses   int `json:"max_uses" binding:"required,min=1"`
	ExpiresIn int `json:"expires_in_hours" binding:"required,min=1"` // Hours until expiration
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
