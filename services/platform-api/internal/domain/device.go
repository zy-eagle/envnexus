package domain

import "time"

type DeviceStatus string

const (
	DeviceStatusOnline      DeviceStatus = "online"
	DeviceStatusOffline     DeviceStatus = "offline"
	DeviceStatusQuarantined DeviceStatus = "quarantined"
	DeviceStatusRevoked     DeviceStatus = "revoked"
)

// Device represents a managed endpoint agent.
type Device struct {
	ID              string
	TenantID        string
	Name            string
	Hostname        string
	OSType          string
	Status          DeviceStatus
	LastHeartbeatAt *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// RecordHeartbeat updates the device's last heartbeat time and status.
func (d *Device) RecordHeartbeat() {
	now := time.Now()
	d.LastHeartbeatAt = &now
	if d.Status == DeviceStatusOffline {
		d.Status = DeviceStatusOnline
	}
	d.UpdatedAt = now
}
