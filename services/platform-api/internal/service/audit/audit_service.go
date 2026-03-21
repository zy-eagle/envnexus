package audit

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/domain"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/dto"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/repository"
)

type Service struct {
	auditRepo repository.AuditRepository
}

func NewService(auditRepo repository.AuditRepository) *Service {
	return &Service{
		auditRepo: auditRepo,
	}
}

func (s *Service) ReportEvent(ctx context.Context, tenantID string, req dto.ReportAuditRequest) (*dto.AuditEventResponse, error) {
	payloadBytes, err := json.Marshal(req.Payload)
	if err != nil {
		payloadBytes = []byte("{}")
	}

	event := &domain.AuditEvent{
		ID:           uuid.New().String(),
		TenantID:     tenantID,
		DeviceID:     req.DeviceID,
		SessionID:    req.SessionID,
		ActionType:   req.ActionType,
		Status:       req.Status,
		Payload:      payloadBytes,
		ErrorMessage: req.ErrorMessage,
		CreatedAt:    time.Now(),
	}

	if err := s.auditRepo.Create(ctx, event); err != nil {
		return nil, err
	}

	return &dto.AuditEventResponse{
		ID:           event.ID,
		TenantID:     event.TenantID,
		DeviceID:     event.DeviceID,
		SessionID:    event.SessionID,
		ActionType:   event.ActionType,
		Status:       event.Status,
		Payload:      req.Payload,
		ErrorMessage: event.ErrorMessage,
		CreatedAt:    event.CreatedAt,
	}, nil
}
