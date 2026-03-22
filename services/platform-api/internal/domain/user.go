package domain

import "time"

type User struct {
	ID           string
	TenantID     string
	Email        string
	PasswordHash string
	DisplayName  string
	Status       string
	LastLoginAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
