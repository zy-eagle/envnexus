package domain

import "time"

type CommandExecutionStatus string

const (
	ExecutionPending   CommandExecutionStatus = "pending"
	ExecutionSent      CommandExecutionStatus = "sent"
	ExecutionRunning   CommandExecutionStatus = "running"
	ExecutionSucceeded CommandExecutionStatus = "succeeded"
	ExecutionFailed    CommandExecutionStatus = "failed"
	ExecutionTimeout   CommandExecutionStatus = "timeout"
	ExecutionSkipped   CommandExecutionStatus = "skipped"
)

type CommandExecution struct {
	ID           string                 `json:"id"             gorm:"primaryKey;size:26"`
	TaskID       string                 `json:"task_id"        gorm:"size:26;not null;index"`
	DeviceID     string                 `json:"device_id"      gorm:"size:26;not null;index"`
	Status       CommandExecutionStatus `json:"status"         gorm:"size:32;not null"`
	Output       *string                `json:"output"         gorm:"type:text"`
	ErrorMessage *string                `json:"error"          gorm:"type:text"`
	ExitCode     *int                   `json:"exit_code"`
	DurationMs   *int                   `json:"duration_ms"`
	SentAt       *time.Time             `json:"sent_at"`
	StartedAt    *time.Time             `json:"started_at"`
	FinishedAt   *time.Time             `json:"finished_at"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

func (c *CommandExecution) TableName() string { return "command_executions" }
