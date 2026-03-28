package package_svc

import (
	"context"
	"fmt"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/domain"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/dto"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/infrastructure"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/repository"
	device_binding "github.com/zy-eagle/envnexus/services/platform-api/internal/service/device_binding"
)

type Service struct {
	pkgRepo     repository.PackageRepository
	minioClient *infrastructure.MinIOClient
}

func NewService(pkgRepo repository.PackageRepository, minioClient *infrastructure.MinIOClient) *Service {
	return &Service{
		pkgRepo:     pkgRepo,
		minioClient: minioClient,
	}
}

func (s *Service) CreatePackage(ctx context.Context, tenantID string, req dto.CreatePackageRequest) (*dto.PackageResponse, error) {
	ext := ""
	if req.Platform == "windows" {
		ext = ".exe"
	}
	pkgName := fmt.Sprintf("envnexus-agent-%s-%s-%s%s", req.Platform, req.Arch, req.Version, ext)
	artifactPath := fmt.Sprintf("packages/%s/%s", tenantID, pkgName)

	activationMode := req.ActivationMode
	if activationMode == "" {
		activationMode = domain.ActivationModeAuto
	}
	maxDevices := req.MaxDevices
	if maxDevices <= 0 {
		maxDevices = 1
	}

	var activationKey string
	var activationKeyHash string
	if activationMode == domain.ActivationModeAuto || activationMode == domain.ActivationModeBoth {
		activationKey = device_binding.GenerateActivationKey()
		activationKeyHash = device_binding.HashActivationKey(activationKey)
	}

	pkg := &domain.DownloadPackage{
		ID:                ulid.Make().String(),
		TenantID:          tenantID,
		AgentProfileID:    req.AgentProfileID,
		DistributionMode:  req.DistributionMode,
		Platform:          req.Platform,
		Arch:              req.Arch,
		Version:           req.Version,
		PackageName:       pkgName,
		DownloadURL:       "",
		ArtifactPath:      artifactPath,
		Checksum:          "",
		SignStatus:        "pending",
		ActivationMode:    activationMode,
		ActivationKeyHash: activationKeyHash,
		MaxDevices:        maxDevices,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	if err := s.pkgRepo.Create(ctx, pkg, activationKey); err != nil {
		return nil, err
	}

	resp := s.toResponse(ctx, pkg)
	if activationKey != "" {
		resp.ActivationKey = activationKey
	}
	return resp, nil
}

func (s *Service) ListPackages(ctx context.Context, tenantID string) ([]*dto.PackageResponse, error) {
	pkgs, err := s.pkgRepo.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	var res []*dto.PackageResponse
	for _, pkg := range pkgs {
		res = append(res, s.toResponse(ctx, pkg))
	}
	return res, nil
}

func (s *Service) DeletePackage(ctx context.Context, tenantID, packageID string) error {
	return s.pkgRepo.Delete(ctx, packageID, tenantID)
}

func (s *Service) GetPresignedURL(ctx context.Context, tenantID, packageID string) (string, error) {
	pkg, err := s.pkgRepo.GetByID(ctx, packageID)
	if err != nil {
		return "", err
	}
	if pkg == nil || pkg.TenantID != tenantID {
		return "", domain.ErrNotFound
	}
	if s.minioClient == nil || pkg.ArtifactPath == "" {
		return "", domain.ErrNotFound
	}

	presignedURL, err := s.minioClient.PresignedGetURL(ctx, pkg.ArtifactPath, 15*time.Minute)
	if err != nil {
		return "", err
	}
	return presignedURL.String(), nil
}

func (s *Service) toResponse(ctx context.Context, pkg *domain.DownloadPackage) *dto.PackageResponse {
	downloadURL := pkg.DownloadURL
	if downloadURL == "" && s.minioClient != nil && pkg.ArtifactPath != "" && pkg.SignStatus == "signed" {
		if u, err := s.minioClient.PresignedGetURL(ctx, pkg.ArtifactPath, 1*time.Hour); err == nil {
			downloadURL = u.String()
		}
	}

	return &dto.PackageResponse{
		ID:               pkg.ID,
		TenantID:         pkg.TenantID,
		AgentProfileID:   pkg.AgentProfileID,
		DistributionMode: pkg.DistributionMode,
		Platform:         pkg.Platform,
		Arch:             pkg.Arch,
		Version:          pkg.Version,
		PackageName:      pkg.PackageName,
		DownloadURL:      downloadURL,
		Checksum:         pkg.Checksum,
		SignStatus:       pkg.SignStatus,
		ActivationMode:   pkg.ActivationMode,
		MaxDevices:       pkg.MaxDevices,
		BoundCount:       pkg.BoundCount,
		CreatedAt:        pkg.CreatedAt,
	}
}

