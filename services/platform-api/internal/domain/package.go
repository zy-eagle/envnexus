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
	PackageType           string `gorm:"column:package_type"`
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
	ActivationMode        string `gorm:"column:activation_mode"`
	ActivationKeyHash     string `gorm:"column:activation_key_hash"`
	MaxDevices            int    `gorm:"column:max_devices"`
	BoundCount            int    `gorm:"column:bound_count"`
	Status                string
	BuildStage            string
	BuildProgress         int
	PublishedAt           *time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

const (
	ActivationModeAuto   = "auto"
	ActivationModeManual = "manual"
	ActivationModeBoth   = "both"
)

func (p *DownloadPackage) SupportsAutoActivation() bool {
	return p.ActivationMode == ActivationModeAuto || p.ActivationMode == ActivationModeBoth
}

func (p *DownloadPackage) SupportsManualBinding() bool {
	return p.ActivationMode == ActivationModeManual || p.ActivationMode == ActivationModeBoth
}

func (p *DownloadPackage) TableName() string { return "download_packages" }
