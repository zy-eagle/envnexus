package domain

import "time"

type DownloadPackage struct {
	ID               string
	TenantID         string
	AgentProfileID   string
	DistributionMode string // "bootstrap_package" or "branded_package"
	Platform         string // "windows", "linux", "darwin"
	Arch             string // "amd64", "arm64"
	Version          string
	PackageName      string
	DownloadURL      string
	ArtifactPath     string
	Checksum         string
	SignStatus       string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
