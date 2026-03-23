package audit

import (
	"context"

	"github.com/zy-eagle/envnexus/services/platform-api/internal/domain"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/dto"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/repository"
)

type Service struct {
	auditRepo repository.AuditRepository
}

func NewService(auditRepo repository.AuditRepository) *Service {
	return &Service{auditRepo: auditRepo}
}

func (s *Service) ListEvents(ctx context.Context, tenantID string, filters repository.AuditFilters) ([]*dto.AuditEventResponse, error) {
	events, err := s.auditRepo.ListByTenant(ctx, tenantID, filters)
	if err != nil {
		return nil, domain.ErrInternalError
	}

	var resp []*dto.AuditEventResponse
	for _, e := range events {
		item := &dto.AuditEventResponse{
			ID:               e.ID,
			TenantID:         e.TenantID,
			EventType:        e.EventType,
			EventPayloadJSON: e.EventPayloadJSON,
			CreatedAt:        e.CreatedAt,
		}
		if e.DeviceID != nil {
			item.DeviceID = *e.DeviceID
		}
		if e.SessionID != nil {
			item.SessionID = *e.SessionID
		}
		resp = append(resp, item)
	}
	return resp, nil
}
