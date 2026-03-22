package dto

import "time"

type PolicyProfileResponse struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	Name       string    `json:"name"`
	PolicyJSON string    `json:"policy_json"`
	Status     string    `json:"status"`
	Version    int       `json:"version"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type CreatePolicyProfileRequest struct {
	Name       string `json:"name" binding:"required"`
	PolicyJSON string `json:"policy_json" binding:"required"`
}

type UpdatePolicyProfileRequest struct {
	Name       string `json:"name"`
	PolicyJSON string `json:"policy_json"`
	Status     string `json:"status"`
}
