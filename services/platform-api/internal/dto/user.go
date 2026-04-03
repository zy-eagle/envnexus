package dto

import "time"

type UserResponse struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	Status      string    `json:"status"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateUserRequest struct {
	Email       string   `json:"email" binding:"required"`
	DisplayName string   `json:"display_name" binding:"required"`
	Password    string   `json:"password"`
	Status      string   `json:"status"`
	RoleIDs     []string `json:"role_ids"`
}

type UpdateUserRequest struct {
	Email       *string `json:"email"`
	DisplayName *string `json:"display_name"`
	Password    *string `json:"password"`
	Status      *string `json:"status"`
}

