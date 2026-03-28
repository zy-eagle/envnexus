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

	pkg := &domain.DownloadPackage{
		ID:               ulid.Make().String(),
		TenantID:         tenantID,
		AgentProfileID:   req.AgentProfileID,
		DistributionMode: req.DistributionMode,
		Platform:         req.Platform,
		Arch:             req.Arch,
		Version:          req.Version,
		PackageName:      pkgName,
		DownloadURL:      "",
		ArtifactPath:     artifactPath,
		Checksum:         "",
		SignStatus:       "pending",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if err := s.pkgRepo.Create(ctx, pkg); err != nil {
		return nil, err
	}

	return s.toResponse(ctx, pkg), nil
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
		CreatedAt:        pkg.CreatedAt,
	}
}

