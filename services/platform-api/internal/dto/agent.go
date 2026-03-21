package dto

type AgentEnrollRequest struct {
	Token    string `json:"token" binding:"required"`
	DeviceID string `json:"device_id" binding:"required"`
	Hostname string `json:"hostname" binding:"required"`
	OSType   string `json:"os_type" binding:"required"`
}

type AgentEnrollResponse struct {
	TenantID    string `json:"tenant_id"`
	DeviceToken string `json:"device_token"`
}
