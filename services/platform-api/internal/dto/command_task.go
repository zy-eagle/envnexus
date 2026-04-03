package dto

import "time"

type CreateCommandTaskRequest struct {
	Title          string   `json:"title" binding:"required"`
	CommandType    string   `json:"command_type" binding:"required,oneof=shell tool"`
	CommandPayload string   `json:"command_payload" binding:"required"`
	DeviceIDs      []string `json:"device_ids" binding:"required,min=1"`
	RiskLevel      string   `json:"risk_level" binding:"required,oneof=L1 L2 L3"`
	Emergency      bool     `json:"emergency"`
	BypassReason   string   `json:"bypass_reason"`
	TargetEnv      string   `json:"target_env"`
	ChangeTicket   string   `json:"change_ticket"`
	BusinessApp    string   `json:"business_app"`
	Note           string   `json:"note"`
}

type UpdateCommandTaskRequest struct {
	Title          string   `json:"title" binding:"required"`
	CommandType    string   `json:"command_type" binding:"required,oneof=shell tool"`
	CommandPayload string   `json:"command_payload" binding:"required"`
	DeviceIDs      []string `json:"device_ids" binding:"required,min=1"`
	RiskLevel      string   `json:"risk_level" binding:"required,oneof=L1 L2 L3"`
	Emergency      bool     `json:"emergency"`
	BypassReason   string   `json:"bypass_reason"`
	TargetEnv      string   `json:"target_env"`
	ChangeTicket   string   `json:"change_ticket"`
	BusinessApp    string   `json:"business_app"`
	Note           string   `json:"note"`
}

type ApproveCommandTaskRequest struct {
	Note string `json:"note"`
}

type DenyCommandTaskRequest struct {
	Reason string `json:"reason"`
}

type CommandTaskResponse struct {
	ID             string                     `json:"id"`
	TenantID       string                     `json:"tenant_id"`
	CreatedBy      string                     `json:"created_by"`
	ApproverID     *string                    `json:"approver_id"`
	ApprovedBy     *string                    `json:"approved_by"`
	PolicySnapshotID *string                  `json:"policy_snapshot_id"`
	Title          string                     `json:"title"`
	CommandType    string                     `json:"command_type"`
	CommandPayload string                     `json:"command_payload"`
	DeviceIDs      []string                   `json:"device_ids"`
	RiskLevel      string                     `json:"risk_level"`
	EffectiveRisk  string                     `json:"effective_risk"`
	BypassApproval bool                       `json:"bypass_approval"`
	Emergency      bool                       `json:"emergency"`
	TargetEnv      string                     `json:"target_env"`
	ChangeTicket   string                     `json:"change_ticket"`
	BusinessApp    string                     `json:"business_app"`
	Note           string                     `json:"note"`
	Status         string                     `json:"status"`
	ApprovalNote   string                     `json:"approval_note"`
	ExpiresAt      time.Time                  `json:"expires_at"`
	ApprovedAt     *time.Time                 `json:"approved_at"`
	CompletedAt    *time.Time                 `json:"completed_at"`
	ArchivedAt     *time.Time                 `json:"archived_at,omitempty"`
	CreatedAt      time.Time                  `json:"created_at"`
	UpdatedAt      time.Time                  `json:"updated_at"`
	Executions     []CommandExecutionResponse `json:"executions,omitempty"`
}

type CommandExecutionResponse struct {
	ID         string     `json:"id"`
	TaskID     string     `json:"task_id"`
	DeviceID   string     `json:"device_id"`
	Status     string     `json:"status"`
	Output     *string    `json:"output"`
	Error      *string    `json:"error"`
	ExitCode   *int       `json:"exit_code"`
	DurationMs *int       `json:"duration_ms"`
	SentAt     *time.Time `json:"sent_at"`
	StartedAt  *time.Time `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

type CommandTaskListResponse struct {
	Tasks []CommandTaskResponse `json:"tasks"`
	Total int64                 `json:"total"`
}

type PendingApprovalCountResponse struct {
	Count int64 `json:"count"`
}
