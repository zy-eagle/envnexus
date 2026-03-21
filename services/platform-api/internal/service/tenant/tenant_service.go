package tenant

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/domain"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/dto"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/repository"
)

type Service struct {
	repo repository.TenantRepository
}

func NewService(repo repository.TenantRepository) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) CreateTenant(ctx context.Context, req dto.CreateTenantRequest) (*dto.TenantResponse, error) {
	// Check if slug exists
	existing, err := s.repo.GetBySlug(ctx, req.Slug)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errors.New("tenant with this slug already exists")
	}

	id := uuid.New().String()
	tenant := domain.NewTenant(id, req.Name, req.Slug)

	if err := s.repo.Create(ctx, tenant); err != nil {
		return nil, err
	}

	return &dto.TenantResponse{
		ID:        tenant.ID,
		Name:      tenant.Name,
		Slug:      tenant.Slug,
		Status:    string(tenant.Status),
		CreatedAt: tenant.CreatedAt,
		UpdatedAt: tenant.UpdatedAt,
	}, nil
}

func (s *Service) GetTenant(ctx context.Context, id string) (*dto.TenantResponse, error) {
	tenant, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if tenant == nil {
		return nil, errors.New("tenant not found")
	}

	return &dto.TenantResponse{
		ID:        tenant.ID,
		Name:      tenant.Name,
		Slug:      tenant.Slug,
		Status:    string(tenant.Status),
		CreatedAt: tenant.CreatedAt,
		UpdatedAt: tenant.UpdatedAt,
	}, nil
}
