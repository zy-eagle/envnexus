package dto

type CreateSessionRequest struct {
	DeviceID       string `json:"device_id" binding:"required"`
	Transport      string `json:"transport" binding:"required"`
	InitiatorType  string `json:"initiator_type" binding:"required"`
	InitialMessage string `json:"initial_message,omitempty"`
}

type ApproveSessionRequest struct {
	ApprovalRequestID string `json:"approval_request_id" binding:"required"`
	Comment           string `json:"comment,omitempty"`
}

type DenySessionRequest struct {
	ApprovalRequestID string `json:"approval_request_id" binding:"required"`
	Reason            string `json:"reason,omitempty"`
}

type AbortSessionRequest struct {
	Reason string `json:"reason,omitempty"`
}

type SessionResponse struct {
	ID            string  `json:"id"`
	TenantID      string  `json:"tenant_id"`
	DeviceID      string  `json:"device_id"`
	Transport     string  `json:"transport"`
	Status        string  `json:"status"`
	InitiatorType string  `json:"initiator_type"`
	StartedAt     string  `json:"started_at"`
	EndedAt       *string `json:"ended_at,omitempty"`
}
