package domain

import "time"

type EnrollmentToken struct {
	ID        string
	TenantID  string
	Token     string
	MaxUses   int
	UsedCount int
	ExpiresAt time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (t *EnrollmentToken) IsValid() bool {
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
