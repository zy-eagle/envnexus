package dto

import "time"

type CreatePackageRequest struct {
	AgentProfileID   string `json:"agent_profile_id" binding:"required"`
	DistributionMode string `json:"distribution_mode" binding:"required"`
	Platform         string `json:"platform" binding:"required"`
	Arch             string `json:"arch" binding:"required"`
	Version          string `json:"version" binding:"required"`
}

type PackageResponse struct {
	ID               string    `json:"id"`
	TenantID         string    `json:"tenant_id"`
	AgentProfileID   string    `json:"agent_profile_id"`
	DistributionMode string    `json:"distribution_mode"`
	Platform         string    `json:"platform"`
	Arch             string    `json:"arch"`
	Version          string    `json:"version"`
	PackageName      string    `json:"package_name"`
	DownloadURL      string    `json:"download_url"`
	Checksum         string    `json:"checksum"`
	SignStatus       string    `json:"sign_status"`
	CreatedAt        time.Time `json:"created_at"`
}
