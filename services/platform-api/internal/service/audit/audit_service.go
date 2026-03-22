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

func (s *Service) ListEvents(ctx context.Context, tenantID string) ([]*dto.AuditEventResponse, error) {
	events, err := s.auditRepo.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	var resp []*dto.AuditEventResponse
	for _, e := range events {
		var payload map[string]interface{}
		if len(e.Payload) > 0 {
			_ = json.Unmarshal(e.Payload, &payload)
		}

		resp = append(resp, &dto.AuditEventResponse{
			ID:           e.ID,
			TenantID:     e.TenantID,
			DeviceID:     e.DeviceID,
			SessionID:    e.SessionID,
			ActionType:   e.ActionType,
			Status:       e.Status,
			Payload:      payload,
			ErrorMessage: e.ErrorMessage,
			CreatedAt:    e.CreatedAt,
		})
	}
	return resp, nil
}
