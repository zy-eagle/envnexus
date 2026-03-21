package package_svc

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/domain"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/dto"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/repository"
)

type Service struct {
	pkgRepo repository.PackageRepository
}

func NewService(pkgRepo repository.PackageRepository) *Service {
	return &Service{
		pkgRepo: pkgRepo,
	}
}

func (s *Service) CreatePackage(ctx context.Context, tenantID string, req dto.CreatePackageRequest) (*dto.PackageResponse, error) {
	pkg := &domain.DownloadPackage{
		ID:               uuid.New().String(),
		TenantID:         tenantID,
		AgentProfileID:   req.AgentProfileID,
		DistributionMode: req.DistributionMode,
		Platform:         req.Platform,
		Arch:             req.Arch,
		Version:          req.Version,
		PackageName:      fmt.Sprintf("envnexus-agent-%s-%s-%s.zip", req.Platform, req.Arch, req.Version),
		DownloadURL:      "", // Would be generated after async build
		ArtifactPath:     "",
		Checksum:         "",
		SignStatus:       "pending",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if err := s.pkgRepo.Create(ctx, pkg); err != nil {
		return nil, err
	}

	// In a real implementation, this would trigger an async job in job-runner to build the package

	return &dto.PackageResponse{
		ID:               pkg.ID,
		TenantID:         pkg.TenantID,
		AgentProfileID:   pkg.AgentProfileID,
		DistributionMode: pkg.DistributionMode,
		Platform:         pkg.Platform,
		Arch:             pkg.Arch,
		Version:          pkg.Version,
		PackageName:      pkg.PackageName,
		DownloadURL:      pkg.DownloadURL,
		Checksum:         pkg.Checksum,
		SignStatus:       pkg.SignStatus,
		CreatedAt:        pkg.CreatedAt,
	}, nil
}

func (s *Service) ListPackages(ctx context.Context, tenantID string) ([]*dto.PackageResponse, error) {
	pkgs, err := s.pkgRepo.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	var res []*dto.PackageResponse
	for _, pkg := range pkgs {
		res = append(res, &dto.PackageResponse{
			ID:               pkg.ID,
			TenantID:         pkg.TenantID,
			AgentProfileID:   pkg.AgentProfileID,
			DistributionMode: pkg.DistributionMode,
			Platform:         pkg.Platform,
			Arch:             pkg.Arch,
			Version:          pkg.Version,
			PackageName:      pkg.PackageName,
			DownloadURL:      pkg.DownloadURL,
			Checksum:         pkg.Checksum,
			SignStatus:       pkg.SignStatus,
			CreatedAt:        pkg.CreatedAt,
		})
	}
	return res, nil
}
