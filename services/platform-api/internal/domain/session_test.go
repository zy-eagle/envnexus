package domain

import (
	"testing"
	"time"
)

func TestSession_CanTransitionTo(t *testing.T) {
	tests := []struct {
		name   string
		from   SessionStatus
		to     SessionStatus
		expect bool
	}{
		{"created->attached", SessionStatusCreated, SessionStatusAttached, true},
		{"created->expired", SessionStatusCreated, SessionStatusExpired, true},
		{"created->diagnosing (invalid)", SessionStatusCreated, SessionStatusDiagnosing, false},
		{"attached->diagnosing", SessionStatusAttached, SessionStatusDiagnosing, true},
		{"attached->expired", SessionStatusAttached, SessionStatusExpired, true},
		{"attached->aborted", SessionStatusAttached, SessionStatusAborted, true},
		{"diagnosing->awaiting_approval", SessionStatusDiagnosing, SessionStatusAwaitingApproval, true},
		{"diagnosing->completed", SessionStatusDiagnosing, SessionStatusCompleted, true},
		{"diagnosing->aborted", SessionStatusDiagnosing, SessionStatusAborted, true},
		{"awaiting_approval->executing", SessionStatusAwaitingApproval, SessionStatusExecuting, true},
		{"awaiting_approval->aborted", SessionStatusAwaitingApproval, SessionStatusAborted, true},
		{"awaiting_approval->completed", SessionStatusAwaitingApproval, SessionStatusCompleted, true},
		{"executing->completed", SessionStatusExecuting, SessionStatusCompleted, true},
		{"executing->aborted", SessionStatusExecuting, SessionStatusAborted, true},
		{"completed->anything (terminal)", SessionStatusCompleted, SessionStatusCreated, false},
		{"aborted->anything (terminal)", SessionStatusAborted, SessionStatusCreated, false},
		{"expired->anything (terminal)", SessionStatusExpired, SessionStatusCreated, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Session{Status: tt.from}
			if got := s.CanTransitionTo(tt.to); got != tt.expect {
				t.Errorf("CanTransitionTo(%s -> %s) = %v, want %v", tt.from, tt.to, got, tt.expect)
			}
		})
	}
}

func TestSession_TransitionTo_SetsEndedAt(t *testing.T) {
	terminals := []SessionStatus{SessionStatusCompleted, SessionStatusAborted, SessionStatusExpired}

	for _, target := range terminals {
		t.Run(string(target), func(t *testing.T) {
			var from SessionStatus
			switch target {
			case SessionStatusCompleted:
				from = SessionStatusDiagnosing
			case SessionStatusAborted:
				from = SessionStatusAttached
			case SessionStatusExpired:
				from = SessionStatusCreated
			}

			s := &Session{Status: from, UpdatedAt: time.Now().Add(-1 * time.Hour)}
			if !s.TransitionTo(target) {
				t.Fatalf("transition %s -> %s should succeed", from, target)
			}
			if s.EndedAt == nil {
				t.Error("expected EndedAt to be set for terminal state")
			}
		})
	}
}

func TestSession_TransitionTo_NonTerminalNoEndedAt(t *testing.T) {
	s := &Session{Status: SessionStatusCreated}
	if !s.TransitionTo(SessionStatusAttached) {
		t.Fatal("expected created->attached to succeed")
	}
	if s.EndedAt != nil {
		t.Error("expected EndedAt to remain nil for non-terminal state")
	}
}

func TestSession_FullLifecycle(t *testing.T) {
	s := &Session{
		Status:    SessionStatusCreated,
		UpdatedAt: time.Now(),
	}

	steps := []SessionStatus{
		SessionStatusAttached,
		SessionStatusDiagnosing,
		SessionStatusAwaitingApproval,
		SessionStatusExecuting,
		SessionStatusCompleted,
	}
	for _, target := range steps {
		if !s.TransitionTo(target) {
			t.Fatalf("transition to %s failed from %s", target, s.Status)
		}
	}
	if s.EndedAt == nil {
		t.Error("expected EndedAt to be set after completed")
	}
}

func TestSession_InvalidTransitionDoesNotMutate(t *testing.T) {
	s := &Session{
		Status:    SessionStatusCompleted,
		UpdatedAt: time.Now().Add(-1 * time.Hour),
	}
	before := s.UpdatedAt

	if s.TransitionTo(SessionStatusCreated) {
		t.Error("expected transition from terminal state to fail")
	}
	if s.Status != SessionStatusCompleted {
		t.Error("status should not change on invalid transition")
	}
	if s.UpdatedAt != before {
		t.Error("UpdatedAt should not change on invalid transition")
	}
}
