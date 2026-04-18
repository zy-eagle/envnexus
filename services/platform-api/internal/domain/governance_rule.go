package domain

import "time"

type GovernanceRule struct {
	ID          string    `json:"id"          gorm:"primaryKey;size:26"`
	TenantID    string    `json:"tenant_id"   gorm:"size:26;not null;index"`
	Name        string    `json:"name"        gorm:"size:255;not null"`
	Description string    `json:"description" gorm:"type:text"`
	RuleType    string    `json:"rule_type"   gorm:"size:64;not null"`
	ConditionJSON string  `json:"condition"   gorm:"column:condition_json;type:text;not null"`
	ActionJSON  string    `json:"action"      gorm:"column:action_json;type:text"`
	Severity    string    `json:"severity"    gorm:"size:16;not null;default:'warning'"`
	Enabled     bool      `json:"enabled"     gorm:"not null;default:true"`
	CreatedBy   string    `json:"created_by"  gorm:"size:26;not null"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (GovernanceRule) TableName() string { return "governance_rules" }

type ToolPermission struct {
	ID         string    `json:"id"          gorm:"primaryKey;size:26"`
	TenantID   string    `json:"tenant_id"   gorm:"size:26;not null;index"`
	ToolName   string    `json:"tool_name"   gorm:"size:128;not null"`
	RoleID     *string   `json:"role_id"     gorm:"size:26"`
	Allowed    bool      `json:"allowed"     gorm:"not null;default:true"`
	MaxRisk    string    `json:"max_risk"    gorm:"size:8"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (ToolPermission) TableName() string { return "tool_permissions" }
