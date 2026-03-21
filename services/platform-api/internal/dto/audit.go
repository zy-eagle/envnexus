package dto

import "time"

type ReportAuditRequest struct {
	DeviceID     string                 `json:"device_id" binding:"required"`
	SessionID    string                 `json:"session_id"`
	ActionType   string                 `json:"action_type" binding:"required"`
	Status       string                 `json:"status" binding:"required"`
	Payload      map[string]interface{} `json:"payload"`
	ErrorMessage string                 `json:"error_message"`
}

type AuditEventResponse struct {
	ID           string                 `json:"id"`
	TenantID     string                 `json:"tenant_id"`
	DeviceID     string                 `json:"device_id"`
	SessionID    string                 `json:"session_id"`
	ActionType   string                 `json:"action_type"`
	Status       string                 `json:"status"`
	Payload      map[string]interface{} `json:"payload"`
	ErrorMessage string                 `json:"error_message"`
	CreatedAt    time.Time              `json:"created_at"`
}
