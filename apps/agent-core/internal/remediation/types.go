package remediation

import (
	"time"
)

type PlanStatus string

const (
	PlanStatusGenerated PlanStatus = "generated"
	PlanStatusApproved  PlanStatus = "approved"
	PlanStatusRejected  PlanStatus = "rejected"
	PlanStatusExecuting PlanStatus = "executing"
	PlanStatusCompleted PlanStatus = "completed"
	PlanStatusFailed    PlanStatus = "failed"
	PlanStatusRolledBack PlanStatus = "rolled_back"
)

type StepStatus string

const (
	StepStatusPending   StepStatus = "pending"
	StepStatusRunning   StepStatus = "running"
	StepStatusSucceeded StepStatus = "succeeded"
	StepStatusFailed    StepStatus = "failed"
	StepStatusSkipped   StepStatus = "skipped"
	StepStatusRolledBack StepStatus = "rolled_back"
	StepStatusAwaitingConfirm StepStatus = "awaiting_confirm"
	StepStatusAwaitingApproval StepStatus = "awaiting_approval"
)

const (
	EventPlanGenerated = "plan_generated"
	EventStepStart     = "step_start"
	EventStepResult    = "step_result"
	EventRollbackStart = "rollback_start"
	EventPlanComplete  = "plan_complete"
	EventStepConfirm   = "step_confirm_required"
	EventStepApproval  = "step_approval_required"
)

const (
	RiskL0 = "L0"
	RiskL1 = "L1"
	RiskL2 = "L2"
	RiskL3 = "L3"
)

type RemediationPlan struct {
	PlanID       string            `json:"plan_id"`
	Summary      string            `json:"summary"`
	RiskLevel    string            `json:"risk_level"`
	Status       PlanStatus        `json:"status"`
	Steps        []RemediationStep `json:"steps"`
	Verification *ToolCheck        `json:"verification,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
}

type RemediationStep struct {
	StepID      int                    `json:"step_id"`
	Description string                 `json:"description"`
	ToolName    string                 `json:"tool_name"`
	Params      map[string]interface{} `json:"params"`
	RiskLevel   string                 `json:"risk_level"`
	DependsOn   []int                  `json:"depends_on,omitempty"`
	Precondition *ToolCheck            `json:"precondition,omitempty"`
	Rollback    *RollbackAction        `json:"rollback,omitempty"`
	Verify      *ToolCheck             `json:"verify,omitempty"`
	Timeout     time.Duration          `json:"timeout,omitempty"`
	Status      StepStatus             `json:"status"`
}

type ToolCheck struct {
	ToolName string                 `json:"tool_name"`
	Params   map[string]interface{} `json:"params"`
	Expect   string                 `json:"expect"`
}

type RollbackAction struct {
	ToolName    string                 `json:"tool_name"`
	Params      map[string]interface{} `json:"params"`
	Description string                 `json:"description"`
}

type VerificationStep struct {
	ToolName string                 `json:"tool_name"`
	Params   map[string]interface{} `json:"params"`
	Expect   string                 `json:"expect"`
}

// OverallRiskLevel returns the highest risk level among all steps.
func (p *RemediationPlan) OverallRiskLevel() string {
	highest := RiskL0
	for _, s := range p.Steps {
		if riskOrd(s.RiskLevel) > riskOrd(highest) {
			highest = s.RiskLevel
		}
	}
	return highest
}

func riskOrd(level string) int {
	switch level {
	case RiskL0:
		return 0
	case RiskL1:
		return 1
	case RiskL2:
		return 2
	case RiskL3:
		return 3
	default:
		return 0
	}
}

// HasWriteOperations returns true if any step uses a non-read-only tool.
func (p *RemediationPlan) HasWriteOperations() bool {
	for _, s := range p.Steps {
		if s.RiskLevel != RiskL0 {
			return true
		}
	}
	return false
}
