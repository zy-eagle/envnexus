package domain

import "time"

type ApprovalRule string

const (
	ApprovalRuleSingle     ApprovalRule = "single"
	ApprovalRuleDual       ApprovalRule = "dual"
	ApprovalRuleSequential ApprovalRule = "sequential"
)

type ApprovalPolicy struct {
	ID                string       `json:"id"                gorm:"primaryKey;size:26"`
	TenantID          string       `json:"tenant_id"         gorm:"size:26;not null;index"`
	Name              string       `json:"name"              gorm:"size:128;not null"`
	RiskLevel         string       `json:"risk_level"        gorm:"size:8;not null"`
	ApproverUserID    *string      `json:"approver_user_id"  gorm:"size:26"`
	ApproverRoleID    *string      `json:"approver_role_id"  gorm:"size:26"`
	AutoApprove       bool         `json:"auto_approve"      gorm:"not null;default:false"`
	ApprovalRule      ApprovalRule `json:"approval_rule"     gorm:"size:32;not null;default:single"`
	SeparationOfDuty  bool         `json:"separation_of_duty" gorm:"not null;default:false"`
	ExpiresMinutes    int          `json:"expires_minutes"   gorm:"not null;default:30"`
	Status            string       `json:"status"            gorm:"size:32;not null;default:active"`
	Priority          int          `json:"priority"          gorm:"not null;default:0"`
	Version           int          `json:"version"           gorm:"not null;default:1"`
	CreatedAt         time.Time    `json:"created_at"`
	UpdatedAt         time.Time    `json:"updated_at"`
}

func (a *ApprovalPolicy) TableName() string { return "approval_policies" }
