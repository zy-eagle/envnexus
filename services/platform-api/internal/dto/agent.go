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
	DeviceID                   string                   `json:"device_id" binding:"required"`
	Status                     string                   `json:"status"`
	AgentVersion               string                   `json:"agent_version"`
	DistributionPackageVersion string                   `json:"distribution_package_version,omitempty"`
	PolicyVersion              int                      `json:"policy_version"`
	Stats                      *Stats                   `json:"stats,omitempty"`
	Environment                *AgentRuntimeEnvironment `json:"environment,omitempty"`
}

// AgentRuntimeEnvironment is reported by agent-core on heartbeat for console/LLM context.
type AgentRuntimeEnvironment struct {
	OSVersion string `json:"os_version,omitempty"`
	Shell     string `json:"shell,omitempty"`
	WorkDir   string `json:"work_dir,omitempty"`
	User      string `json:"user,omitempty"`
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

type CheckUpdateResponse struct {
	HasUpdate      bool   `json:"has_update"`
	CurrentVersion string `json:"current_version,omitempty"`
	LatestVersion  string `json:"latest_version,omitempty"`
	DownloadURL    string `json:"download_url,omitempty"`
	PackageID      string `json:"package_id,omitempty"`
	Checksum       string `json:"checksum,omitempty"`
	ArtifactSize   int64  `json:"artifact_size,omitempty"`
	Message        string `json:"message,omitempty"`
}

type AgentAuditEventsRequest struct {
	Events []AgentAuditEventItem `json:"events" binding:"required"`
}

type AgentAuditEventItem struct {
	EventType    string      `json:"event_type" binding:"required"`
	SessionID    string      `json:"session_id,omitempty"`
	EventPayload interface{} `json:"event_payload"`
}
