package domain

import "time"

type SessionStatus string

const (
	SessionStatusCreated          SessionStatus = "created"
	SessionStatusAttached         SessionStatus = "attached"
	SessionStatusDiagnosing       SessionStatus = "diagnosing"
	SessionStatusAwaitingApproval SessionStatus = "awaiting_approval"
	SessionStatusExecuting        SessionStatus = "executing"
	SessionStatusCompleted        SessionStatus = "completed"
	SessionStatusAborted          SessionStatus = "aborted"
	SessionStatusExpired          SessionStatus = "expired"
)

type Session struct {
	ID            string
	TenantID      string
	DeviceID      string
	Transport     string
	Status        SessionStatus
	InitiatorType string
	StartedAt     time.Time
	EndedAt       *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (s *Session) TableName() string { return "sessions" }

func (s *Session) CanTransitionTo(target SessionStatus) bool {
	allowed := map[SessionStatus][]SessionStatus{
		SessionStatusCreated:          {SessionStatusAttached, SessionStatusExpired},
		SessionStatusAttached:         {SessionStatusDiagnosing, SessionStatusExpired, SessionStatusAborted},
		SessionStatusDiagnosing:       {SessionStatusAwaitingApproval, SessionStatusCompleted, SessionStatusAborted},
		SessionStatusAwaitingApproval: {SessionStatusExecuting, SessionStatusAborted, SessionStatusCompleted},
		SessionStatusExecuting:        {SessionStatusCompleted, SessionStatusAborted},
	}
	targets, ok := allowed[s.Status]
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

func (s *Session) TransitionTo(target SessionStatus) bool {
	if !s.CanTransitionTo(target) {
		return false
	}
	s.Status = target
	s.UpdatedAt = time.Now()
	if target == SessionStatusCompleted || target == SessionStatusAborted || target == SessionStatusExpired {
		now := time.Now()
		s.EndedAt = &now
	}
	return true
}
