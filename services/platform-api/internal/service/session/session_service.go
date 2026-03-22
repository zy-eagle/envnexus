package session

import (
	"context"

	"github.com/zy-eagle/envnexus/services/platform-api/internal/dto"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/repository"
)

type Service struct {
	repo repository.SessionRepository
}

func NewService(repo repository.SessionRepository) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) ListSessions(ctx context.Context, tenantID string) ([]*dto.SessionResponse, error) {
	sessions, err := s.repo.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	var resp []*dto.SessionResponse
	for _, sess := range sessions {
		resp = append(resp, &dto.SessionResponse{
			ID:            sess.ID,
			TenantID:      sess.TenantID,
			DeviceID:      sess.DeviceID,
			Transport:     sess.Transport,
			Status:        sess.Status,
			InitiatorType: sess.InitiatorType,
			StartedAt:     sess.StartedAt,
			EndedAt:       sess.EndedAt,
			CreatedAt:     sess.CreatedAt,
			UpdatedAt:     sess.UpdatedAt,
		})
	}
	return resp, nil
}
