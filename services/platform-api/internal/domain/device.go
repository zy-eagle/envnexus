package domain

import (
	"strings"
	"time"
)

type DeviceStatus string

const (
	DeviceStatusDownloaded        DeviceStatus = "downloaded"
	DeviceStatusBootstrapping     DeviceStatus = "bootstrapping"
	DeviceStatusPendingActivation DeviceStatus = "pending_activation"
	DeviceStatusActive            DeviceStatus = "active"
	DeviceStatusPolicyOutdated    DeviceStatus = "policy_outdated"
	DeviceStatusQuarantined       DeviceStatus = "quarantined"
	DeviceStatusRevoked           DeviceStatus = "revoked"
	DeviceStatusRetired           DeviceStatus = "retired"
)

type Device struct {
	ID                         string
	TenantID                   string
	AgentProfileID             string
	DeviceName                 string
	Hostname                   *string
	Platform                   string
	Arch                       string
	RuntimeMetadata            *string `gorm:"column:runtime_metadata;type:json"`
	EnvironmentType            string
	AgentVersion               string
	DistributionPackageVersion string `gorm:"column:distribution_package_version"`
	Status                     DeviceStatus
	PolicyVersion              int
	LastSeenAt                 *time.Time
	CreatedAt                  time.Time
	UpdatedAt                  time.Time
	DeletedAt                  *time.Time
}

func (d *Device) TableName() string { return "devices" }

func (d *Device) RecordHeartbeat(agentVersion, distPkgVersion string, policyVersion int) {
	now := time.Now()
	d.LastSeenAt = &now
	d.AgentVersion = agentVersion
	if distPkgVersion != "" {
		d.DistributionPackageVersion = strings.TrimPrefix(strings.TrimPrefix(distPkgVersion, "v"), "V")
	}
	d.PolicyVersion = policyVersion
	if d.Status == DeviceStatusPendingActivation {
		d.Status = DeviceStatusActive
	}
	d.UpdatedAt = now
}

func (d *Device) IsRevoked() bool {
	return d.Status == DeviceStatusRevoked || d.Status == DeviceStatusRetired
}
