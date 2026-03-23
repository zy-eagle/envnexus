package domain

import "time"

type ApprovalStatus string

const (
	ApprovalStatusDrafted     ApprovalStatus = "drafted"
	ApprovalStatusPendingUser ApprovalStatus = "pending_user"
	ApprovalStatusApproved    ApprovalStatus = "approved"
	ApprovalStatusDenied      ApprovalStatus = "denied"
	ApprovalStatusExpired     ApprovalStatus = "expired"
	ApprovalStatusExecuting   ApprovalStatus = "executing"
	ApprovalStatusSucceeded   ApprovalStatus = "succeeded"
	ApprovalStatusFailed      ApprovalStatus = "failed"
	ApprovalStatusRolledBack  ApprovalStatus = "rolled_back"
)

type ApprovalRequest struct {
	ID                  string
	SessionID           string
	DeviceID            string
	RequestedActionJSON string
	RiskLevel           string
	Status              ApprovalStatus
	ApproverUserID      *string
	ApprovedAt          *time.Time
	ExpiresAt           *time.Time
	ExecutedAt          *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func (a *ApprovalRequest) CanApprove() bool {
	return a.Status == ApprovalStatusPendingUser
}

func (a *ApprovalRequest) CanDeny() bool {
	return a.Status == ApprovalStatusPendingUser
}

func (a *ApprovalRequest) IsExpired() bool {
	if a.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*a.ExpiresAt)
}
