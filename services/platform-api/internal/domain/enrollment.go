package domain

import "time"

type EnrollmentToken struct {
	ID                string
	TenantID          string
	AgentProfileID    string
	DownloadPackageID string
	TokenHash         string
	Channel           string
	ExpiresAt         time.Time
	MaxUses           int
	UsedCount         int
	IssuedByUserID    *string
	Status            string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (t *EnrollmentToken) TableName() string { return "enrollment_tokens" }

func (t *EnrollmentToken) IsValid() bool {
	if t.Status != "active" {
		return false
	}
	if t.UsedCount >= t.MaxUses {
		return false
	}
	if time.Now().After(t.ExpiresAt) {
		return false
	}
	return true
}

func (t *EnrollmentToken) IncrementUsage() {
	t.UsedCount++
	t.UpdatedAt = time.Now()
}
