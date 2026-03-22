package model_profile

import (
	"context"

	"github.com/google/uuid"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/domain"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/dto"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/repository"
)

type Service struct {
	repo repository.ModelProfileRepository
}

func NewService(repo repository.ModelProfileRepository) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) ListProfiles(ctx context.Context, tenantID string) ([]*dto.ModelProfileResponse, error) {
	profiles, err := s.repo.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	var resp []*dto.ModelProfileResponse
	for _, p := range profiles {
		resp = append(resp, &dto.ModelProfileResponse{
			ID:         p.ID,
			TenantID:   p.TenantID,
			Name:       p.Name,
			Provider:   p.Provider,
			BaseURL:    p.BaseURL,
			ModelName:  p.ModelName,
			ParamsJSON: p.ParamsJSON,
			SecretMode: p.SecretMode,
			Status:     p.Status,
			Version:    p.Version,
			CreatedAt:  p.CreatedAt,
			UpdatedAt:  p.UpdatedAt,
		})
	}
	return resp, nil
}

func (s *Service) CreateProfile(ctx context.Context, tenantID string, req dto.CreateModelProfileRequest) (*dto.ModelProfileResponse, error) {
	id := uuid.New().String()
	profile := &domain.ModelProfile{
		ID:         id,
		TenantID:   tenantID,
		Name:       req.Name,
		Provider:   req.Provider,
		BaseURL:    req.BaseURL,
		ModelName:  req.ModelName,
		ParamsJSON: req.ParamsJSON,
		SecretMode: req.SecretMode,
		Status:     "active",
		Version:    1,
	}

	if err := s.repo.Create(ctx, profile); err != nil {
		return nil, err
	}

	return &dto.ModelProfileResponse{
		ID:         profile.ID,
		TenantID:   profile.TenantID,
		Name:       profile.Name,
		Provider:   profile.Provider,
		BaseURL:    profile.BaseURL,
		ModelName:  profile.ModelName,
		ParamsJSON: profile.ParamsJSON,
		SecretMode: profile.SecretMode,
		Status:     profile.Status,
		Version:    profile.Version,
		CreatedAt:  profile.CreatedAt,
		UpdatedAt:  profile.UpdatedAt,
	}, nil
}
