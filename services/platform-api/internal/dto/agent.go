package dto

type AgentEnrollRequest struct {
	EnrollmentToken string           `json:"enrollment_token" binding:"required"`
	Device          AgentDeviceInfo  `json:"device" binding:"required"`
	Agent           AgentVersionInfo `json:"agent" binding:"required"`
}

type AgentDeviceInfo struct {
	DeviceName      string `json:"device_name" binding:"required"`
	Hostname        string `json:"hostname"`
	Platform        string `json:"platform" binding:"required"`
	Arch            string `json:"arch"`
	EnvironmentType string `json:"environment_type"`
}

type AgentVersionInfo struct {
	Version string `json:"version" binding:"required"`
}

type AgentHeartbeatRequest struct {
	DeviceID      string  `json:"device_id" binding:"required"`
	Status        string  `json:"status"`
	AgentVersion  string  `json:"agent_version"`
	PolicyVersion int     `json:"policy_version"`
	Stats         *Stats  `json:"stats,omitempty"`
}

type Stats struct {
	CPUPercent float64 `json:"cpu_percent"`
	MemoryMB   int     `json:"memory_mb"`
}

type AgentConfigResponse struct {
	HasUpdate     bool        `json:"has_update"`
	ConfigVersion int         `json:"config_version"`
	AgentProfile  interface{} `json:"agent_profile,omitempty"`
	ModelProfile  interface{} `json:"model_profile,omitempty"`
	PolicyProfile interface{} `json:"policy_profile,omitempty"`
}

type AgentAuditEventsRequest struct {
	Events []AgentAuditEventItem `json:"events" binding:"required"`
}

type AgentAuditEventItem struct {
	EventType    string      `json:"event_type" binding:"required"`
	SessionID    string      `json:"session_id,omitempty"`
	EventPayload interface{} `json:"event_payload"`
}
