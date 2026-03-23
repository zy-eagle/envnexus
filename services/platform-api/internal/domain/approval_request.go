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

func (a *ApprovalRequest) TableName() string { return "approval_requests" }

func (a *ApprovalRequest) CanApprove() bool {
	return a.Status == ApprovalStatusPendingUser
}

func (a *ApprovalRequest) CanDeny() bool {
	return a.Status == ApprovalStatusPendingUser
}

func (a *ApprovalRequest) CanTransitionTo(target ApprovalStatus) bool {
	allowed := map[ApprovalStatus][]ApprovalStatus{
		ApprovalStatusDrafted:     {ApprovalStatusPendingUser},
		ApprovalStatusPendingUser: {ApprovalStatusApproved, ApprovalStatusDenied, ApprovalStatusExpired},
		ApprovalStatusApproved:    {ApprovalStatusExecuting},
		ApprovalStatusExecuting:   {ApprovalStatusSucceeded, ApprovalStatusFailed},
		ApprovalStatusFailed:      {ApprovalStatusRolledBack},
	}
	targets, ok := allowed[a.Status]
	if !ok {
		return false
	}
	for _, t := range targets {
		if t == target {
			return true
		}
	}
	return false
}

func (a *ApprovalRequest) TransitionTo(target ApprovalStatus) bool {
	if !a.CanTransitionTo(target) {
		return false
	}
	a.Status = target
	a.UpdatedAt = time.Now()
	return true
}

func (a *ApprovalRequest) IsExpired() bool {
	if a.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*a.ExpiresAt)
}

func (a *ApprovalRequest) IsTerminal() bool {
	return a.Status == ApprovalStatusDenied ||
		a.Status == ApprovalStatusExpired ||
		a.Status == ApprovalStatusSucceeded ||
		a.Status == ApprovalStatusRolledBack
}
