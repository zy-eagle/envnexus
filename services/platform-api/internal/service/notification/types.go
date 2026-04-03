package notification

import "time"

type ApprovalNotification struct {
	TaskID      string
	TenantID    string
	TargetUser  string
	RequestedBy string
	Title       string
	CommandType string
	RiskLevel   string
	DeviceCount int
	ExpiresAt   time.Time
}

type EmergencyNotification struct {
	TaskID         string
	TenantID       string
	TargetUser     string
	RequestedBy    string
	Title          string
	CommandType    string
	CommandPayload string
	RiskLevel      string
	DeviceCount    int
	BypassReason   string
}

type ResultNotification struct {
	TaskID      string
	TenantID    string
	TargetUser  string
	Title       string
	CommandType string
	RiskLevel   string
	Status      string
	DeviceCount int
	Succeeded   int
	Failed      int
}

type Notifier interface {
	ProviderType() string
	SendApprovalCard(taskID, tenantID, targetUserExternalID, requesterName, title, commandType, riskLevel string, deviceCount int, expiresAt time.Time) error
	SendEmergencyCard(taskID, tenantID, targetUserExternalID, requesterName, title, command, riskLevel string, deviceCount int, reason string) error
	SendResultCard(taskID, tenantID, targetUserExternalID, title, status string, succeeded, failed int) error
}
