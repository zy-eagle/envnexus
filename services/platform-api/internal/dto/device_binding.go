package dto

import "time"

// --- Agent-side requests/responses ---

type ComponentInfo struct {
	Type string `json:"type" binding:"required"` // cpu, board, mac, disk, gpu
	Hash string `json:"hash" binding:"required"`
}

type DeviceInfo struct {
	OS       string `json:"os"`
	Hostname string `json:"hostname"`
	CPUModel string `json:"cpu_model"`
}

type RegisterDeviceRequest struct {
	Components []ComponentInfo `json:"components" binding:"required,min=1"`
	DeviceInfo *DeviceInfo     `json:"device_info"`
}

type RegisterDeviceResponse struct {
	DeviceCode string `json:"device_code"`
}

type ActivateDeviceRequest struct {
	ActivationKey string          `json:"activation_key" binding:"required"`
	DeviceCode    string          `json:"device_code" binding:"required"`
	Components    []ComponentInfo `json:"components" binding:"required,min=1"`
}

type ActivateDeviceResponse struct {
	Activated bool   `json:"activated"`
	PackageID string `json:"package_id,omitempty"`
	TenantID  string `json:"tenant_id,omitempty"`
	Error     string `json:"error,omitempty"`
}

type ActivationStatusResponse struct {
	Activated      bool   `json:"activated"`
	PackageID      string `json:"package_id,omitempty"`
	TenantID       string `json:"tenant_id,omitempty"`
	ActivationMode string `json:"activation_mode,omitempty"`
}

// --- Admin-side requests/responses ---

type BindDeviceRequest struct {
	DeviceCode string `json:"device_code" binding:"required"`
}

type BindDeviceResponse struct {
	BindingID  string    `json:"binding_id"`
	DeviceCode string    `json:"device_code"`
	Status     string    `json:"status"`
	BoundAt    time.Time `json:"bound_at"`
}

type UpdateMaxDevicesRequest struct {
	MaxDevices int `json:"max_devices" binding:"required,min=1"`
}

type DeviceBindingResponse struct {
	ID            string     `json:"id"`
	DeviceCode    string     `json:"device_code"`
	DeviceInfo    *DeviceInfo `json:"device_info,omitempty"`
	Status        string     `json:"status"`
	BoundBy       string     `json:"bound_by"`
	BoundAt       time.Time  `json:"bound_at"`
	LastHeartbeat *time.Time `json:"last_heartbeat,omitempty"`
}

type ActivationAuditLogResponse struct {
	ID         string     `json:"id"`
	PackageID  string     `json:"package_id"`
	DeviceCode string     `json:"device_code"`
	Action     string     `json:"action"`
	Actor      string     `json:"actor"`
	Detail     *string    `json:"detail,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// --- Heartbeat ---

type HeartbeatRequest struct {
	DeviceCode string          `json:"device_code" binding:"required"`
	Components []ComponentInfo `json:"components"`
}

type HeartbeatResponse struct {
	Status string `json:"status"` // ok, revoked
}
