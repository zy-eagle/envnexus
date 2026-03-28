package domain

import "time"

type DeviceBinding struct {
	ID            string
	TenantID      string
	PackageID     string
	DeviceCode    string
	HardwareHash  string
	DeviceInfo    *string
	Status        string // active, revoked
	BoundAt       time.Time
	BoundBy       string // admin user ID or "system"
	RevokedAt     *time.Time
	LastHeartbeat *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (d *DeviceBinding) TableName() string { return "device_bindings" }

const (
	BindingStatusActive  = "active"
	BindingStatusRevoked = "revoked"
)

type DeviceComponent struct {
	ID            string
	DeviceCode    string
	ComponentType string // cpu, board, mac, disk, gpu
	ComponentHash string
	CreatedAt     time.Time
}

func (c *DeviceComponent) TableName() string { return "device_components" }

const (
	ComponentCPU   = "cpu"
	ComponentBoard = "board"
	ComponentMAC   = "mac"
	ComponentDisk  = "disk"
	ComponentGPU   = "gpu"
)

type PendingDevice struct {
	ID           string
	DeviceCode   string
	HardwareHash string
	DeviceInfo   *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (p *PendingDevice) TableName() string { return "pending_devices" }

type ActivationAuditLog struct {
	ID         string
	TenantID   string
	PackageID  string
	DeviceCode string
	Action     string // activate, bind, unbind, revoke, heartbeat_fail
	Actor      string
	Detail     *string
	CreatedAt  time.Time
}

func (a *ActivationAuditLog) TableName() string { return "activation_audit_logs" }

const (
	AuditActionActivate     = "activate"
	AuditActionBind         = "bind"
	AuditActionUnbind       = "unbind"
	AuditActionRevoke       = "revoke"
	AuditActionHeartbeatFail = "heartbeat_fail"
)
