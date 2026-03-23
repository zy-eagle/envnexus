package domain

import "time"

type DownloadPackage struct {
	ID                    string
	TenantID              string
	AgentProfileID        string
	DistributionMode      string
	Platform              string
	Arch                  string
	Version               string
	PackageName           string
	DownloadURL           string
	ArtifactPath          string
	ArtifactSize          int64
	Checksum              string
	BootstrapManifestJSON *string
	BrandingVersion       int
	BuildVersion          string
	SignStatus            string
	SignMetadataJSON      *string
	Status                string
	PublishedAt           *time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

func (p *DownloadPackage) TableName() string { return "download_packages" }
