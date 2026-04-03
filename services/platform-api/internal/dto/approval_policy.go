package dto

import "time"

type CreateApprovalPolicyRequest struct {
	Name             string `json:"name" binding:"required"`
	RiskLevel        string `json:"risk_level" binding:"required,oneof=L1 L2 L3 *"`
	ApproverUserID   string `json:"approver_user_id"`
	ApproverRoleID   string `json:"approver_role_id"`
	AutoApprove      bool   `json:"auto_approve"`
	ApprovalRule     string `json:"approval_rule" binding:"omitempty,oneof=single dual sequential"`
	SeparationOfDuty bool   `json:"separation_of_duty"`
	ExpiresMinutes   int    `json:"expires_minutes"`
	Priority         int    `json:"priority"`
}

type UpdateApprovalPolicyRequest struct {
	Name             *string `json:"name"`
	ApproverUserID   *string `json:"approver_user_id"`
	ApproverRoleID   *string `json:"approver_role_id"`
	AutoApprove      *bool   `json:"auto_approve"`
	ApprovalRule     *string `json:"approval_rule"`
	SeparationOfDuty *bool   `json:"separation_of_duty"`
	ExpiresMinutes   *int    `json:"expires_minutes"`
	Priority         *int    `json:"priority"`
	Status           *string `json:"status"`
}

type ApprovalPolicyResponse struct {
	ID               string    `json:"id"`
	TenantID         string    `json:"tenant_id"`
	Name             string    `json:"name"`
	RiskLevel        string    `json:"risk_level"`
	ApproverUserID   *string   `json:"approver_user_id"`
	ApproverRoleID   *string   `json:"approver_role_id"`
	AutoApprove      bool      `json:"auto_approve"`
	ApprovalRule     string    `json:"approval_rule"`
	SeparationOfDuty bool      `json:"separation_of_duty"`
	ExpiresMinutes   int       `json:"expires_minutes"`
	Status           string    `json:"status"`
	Priority         int       `json:"priority"`
	Version          int       `json:"version"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
