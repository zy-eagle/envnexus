package domain

import (
	"encoding/json"
	"time"
)

type CommandTaskStatus string

const (
	CommandTaskPendingApproval CommandTaskStatus = "pending_approval"
	CommandTaskApproved        CommandTaskStatus = "approved"
	CommandTaskDenied          CommandTaskStatus = "denied"
	CommandTaskExecuting       CommandTaskStatus = "executing"
	CommandTaskPartialDone     CommandTaskStatus = "partial_done"
	CommandTaskCompleted       CommandTaskStatus = "completed"
	CommandTaskFailed          CommandTaskStatus = "failed"
	CommandTaskExpired         CommandTaskStatus = "expired"
	CommandTaskCancelled       CommandTaskStatus = "cancelled"
)

type CommandTask struct {
	ID                string            `json:"id"                gorm:"primaryKey;size:26"`
	TenantID          string            `json:"tenant_id"         gorm:"size:26;not null;index"`
	CreatedByUserID   string            `json:"created_by"        gorm:"size:26;not null"`
	ApproverUserID    *string           `json:"approver_id"       gorm:"size:26"`
	ApproverRoleID    *string           `json:"approver_role_id"  gorm:"size:26"`
	ApprovedByID      *string           `json:"approved_by"       gorm:"size:26"`
	Title             string            `json:"title"             gorm:"size:255;not null"`
	CommandType       string            `json:"command_type"      gorm:"size:64;not null"`
	CommandPayload    string            `json:"command_payload"   gorm:"type:text;not null"`
	DeviceIDsJSON     string            `json:"device_ids"        gorm:"column:device_ids;type:text;not null"`
	RiskLevel         string            `json:"risk_level"        gorm:"size:8;not null"`
	EffectiveRisk     string            `json:"effective_risk"    gorm:"size:8;not null"`
	BypassApproval    bool              `json:"bypass_approval"   gorm:"not null;default:false"`
	BypassReason      string            `json:"bypass_reason"     gorm:"type:text"`
	Emergency         bool              `json:"emergency"         gorm:"not null;default:false"`
	PolicySnapshotID  *string           `json:"policy_snapshot_id" gorm:"size:26"`
	TargetEnv         string            `json:"target_env"        gorm:"size:32"`
	ChangeTicket      string            `json:"change_ticket"     gorm:"size:255"`
	BusinessApp       string            `json:"business_app"      gorm:"size:255"`
	Note              string            `json:"note"              gorm:"type:text"`
	Status            CommandTaskStatus `json:"status"            gorm:"size:32;not null"`
	ApprovalNote      string            `json:"approval_note"     gorm:"type:text"`
	ExpiresAt         time.Time         `json:"expires_at"        gorm:"not null"`
	ApprovedAt        *time.Time        `json:"approved_at"`
	CompletedAt       *time.Time        `json:"completed_at"`
	ArchivedAt       *time.Time        `json:"archived_at"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
}

func (c *CommandTask) TableName() string { return "command_tasks" }

func (c *CommandTask) ParseDeviceIDs() []string {
	var ids []string
	_ = json.Unmarshal([]byte(c.DeviceIDsJSON), &ids)
	return ids
}

func (c *CommandTask) SetDeviceIDs(ids []string) {
	data, _ := json.Marshal(ids)
	c.DeviceIDsJSON = string(data)
}
