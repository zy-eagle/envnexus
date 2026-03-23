package domain

import "time"

type ToolInvocation struct {
	ID         string
	SessionID  string
	DeviceID   string
	ToolName   string
	RiskLevel  string
	InputJSON  string
	OutputJSON *string
	Status     string
	DurationMs *int
	StartedAt  time.Time
	FinishedAt *time.Time
	CreatedAt  time.Time
}
