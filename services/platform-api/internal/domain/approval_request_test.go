package domain

import (
	"testing"
	"time"
)

func TestApprovalRequest_CanTransitionTo(t *testing.T) {
	tests := []struct {
		name   string
		from   ApprovalStatus
		to     ApprovalStatus
		expect bool
	}{
		{"drafted->pending_user", ApprovalStatusDrafted, ApprovalStatusPendingUser, true},
		{"drafted->approved (invalid)", ApprovalStatusDrafted, ApprovalStatusApproved, false},
		{"pending_user->approved", ApprovalStatusPendingUser, ApprovalStatusApproved, true},
		{"pending_user->denied", ApprovalStatusPendingUser, ApprovalStatusDenied, true},
		{"pending_user->expired", ApprovalStatusPendingUser, ApprovalStatusExpired, true},
		{"pending_user->executing (invalid)", ApprovalStatusPendingUser, ApprovalStatusExecuting, false},
		{"approved->executing", ApprovalStatusApproved, ApprovalStatusExecuting, true},
		{"approved->denied (invalid)", ApprovalStatusApproved, ApprovalStatusDenied, false},
		{"executing->succeeded", ApprovalStatusExecuting, ApprovalStatusSucceeded, true},
		{"executing->failed", ApprovalStatusExecuting, ApprovalStatusFailed, true},
		{"executing->rolled_back (invalid)", ApprovalStatusExecuting, ApprovalStatusRolledBack, false},
		{"failed->rolled_back", ApprovalStatusFailed, ApprovalStatusRolledBack, true},
		{"succeeded->anything (terminal)", ApprovalStatusSucceeded, ApprovalStatusDrafted, false},
		{"denied->anything (terminal)", ApprovalStatusDenied, ApprovalStatusApproved, false},
		{"expired->anything (terminal)", ApprovalStatusExpired, ApprovalStatusPendingUser, false},
		{"rolled_back->anything (terminal)", ApprovalStatusRolledBack, ApprovalStatusDrafted, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &ApprovalRequest{Status: tt.from}
			if got := a.CanTransitionTo(tt.to); got != tt.expect {
				t.Errorf("CanTransitionTo(%s -> %s) = %v, want %v", tt.from, tt.to, got, tt.expect)
			}
		})
	}
}

func TestApprovalRequest_TransitionTo(t *testing.T) {
	a := &ApprovalRequest{
		Status:    ApprovalStatusDrafted,
		UpdatedAt: time.Now().Add(-1 * time.Hour),
	}
	before := a.UpdatedAt

	if !a.TransitionTo(ApprovalStatusPendingUser) {
		t.Fatal("expected transition drafted->pending_user to succeed")
	}
	if a.Status != ApprovalStatusPendingUser {
		t.Errorf("status = %s, want %s", a.Status, ApprovalStatusPendingUser)
	}
	if !a.UpdatedAt.After(before) {
		t.Error("expected UpdatedAt to be advanced")
	}

	if a.TransitionTo(ApprovalStatusExecuting) {
		t.Error("expected transition pending_user->executing to fail")
	}
}

func TestApprovalRequest_CanApprove_CanDeny(t *testing.T) {
	a := &ApprovalRequest{Status: ApprovalStatusPendingUser}
	if !a.CanApprove() {
		t.Error("expected CanApprove=true for pending_user")
	}
	if !a.CanDeny() {
		t.Error("expected CanDeny=true for pending_user")
	}

	a.Status = ApprovalStatusApproved
	if a.CanApprove() {
		t.Error("expected CanApprove=false for approved")
	}
}

func TestApprovalRequest_IsExpired(t *testing.T) {
	a := &ApprovalRequest{}
	if a.IsExpired() {
		t.Error("nil ExpiresAt should not be expired")
	}

	past := time.Now().Add(-1 * time.Minute)
	a.ExpiresAt = &past
	if !a.IsExpired() {
		t.Error("past ExpiresAt should be expired")
	}

	future := time.Now().Add(1 * time.Hour)
	a.ExpiresAt = &future
	if a.IsExpired() {
		t.Error("future ExpiresAt should not be expired")
	}
}

func TestApprovalRequest_IsTerminal(t *testing.T) {
	terminals := []ApprovalStatus{
		ApprovalStatusDenied, ApprovalStatusExpired,
		ApprovalStatusSucceeded, ApprovalStatusRolledBack,
	}
	for _, s := range terminals {
		a := &ApprovalRequest{Status: s}
		if !a.IsTerminal() {
			t.Errorf("expected %s to be terminal", s)
		}
	}

	nonTerminals := []ApprovalStatus{
		ApprovalStatusDrafted, ApprovalStatusPendingUser,
		ApprovalStatusApproved, ApprovalStatusExecuting, ApprovalStatusFailed,
	}
	for _, s := range nonTerminals {
		a := &ApprovalRequest{Status: s}
		if a.IsTerminal() {
			t.Errorf("expected %s to NOT be terminal", s)
		}
	}
}

func TestApprovalRequest_FullLifecycle(t *testing.T) {
	a := &ApprovalRequest{Status: ApprovalStatusDrafted}

	steps := []ApprovalStatus{
		ApprovalStatusPendingUser,
		ApprovalStatusApproved,
		ApprovalStatusExecuting,
		ApprovalStatusSucceeded,
	}
	for _, target := range steps {
		if !a.TransitionTo(target) {
			t.Fatalf("transition to %s failed from %s", target, a.Status)
		}
	}
	if !a.IsTerminal() {
		t.Error("expected succeeded to be terminal")
	}
}

func TestApprovalRequest_FailedRollbackPath(t *testing.T) {
	a := &ApprovalRequest{Status: ApprovalStatusDrafted}
	a.TransitionTo(ApprovalStatusPendingUser)
	a.TransitionTo(ApprovalStatusApproved)
	a.TransitionTo(ApprovalStatusExecuting)
	a.TransitionTo(ApprovalStatusFailed)

	if !a.TransitionTo(ApprovalStatusRolledBack) {
		t.Fatal("expected failed->rolled_back to succeed")
	}
	if !a.IsTerminal() {
		t.Error("expected rolled_back to be terminal")
	}
}
