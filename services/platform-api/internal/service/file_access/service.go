package file_access

import (
	"context"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/zy-eagle/envnexus/services/platform-api/internal/domain"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/repository"
)

type Service struct {
	repo     repository.FileAccessRepository
	auditRepo repository.AuditRepository
}

func NewService(repo repository.FileAccessRepository, auditRepo repository.AuditRepository) *Service {
	return &Service{repo: repo, auditRepo: auditRepo}
}

func (s *Service) CreateRequest(ctx context.Context, tenantID, deviceID, requestedBy, path string, action domain.FileAccessAction, note string) (*domain.FileAccessRequest, error) {
	req := &domain.FileAccessRequest{
		ID:          ulid.Make().String(),
		TenantID:    tenantID,
		DeviceID:    deviceID,
		RequestedBy: requestedBy,
		Path:        path,
		Action:      action,
		Status:      domain.FileAccessPending,
		Note:        note,
		ExpiresAt:   time.Now().Add(24 * time.Hour),
	}
	if err := s.repo.Create(ctx, req); err != nil {
		return nil, err
	}
	return req, nil
}

func (s *Service) Approve(ctx context.Context, id, approverID string) (*domain.FileAccessRequest, error) {
	req, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	if req.Status != domain.FileAccessPending {
		return nil, domain.NewAppError("invalid_state", "request is not pending", 409)
	}
	if time.Now().After(req.ExpiresAt) {
		req.Status = domain.FileAccessExpired
		_ = s.repo.Update(ctx, req)
		return nil, domain.NewAppError("expired", "file access request has expired", 409)
	}
	now := time.Now()
	req.Status = domain.FileAccessApproved
	req.ApprovedBy = &approverID
	req.ResolvedAt = &now
	if err := s.repo.Update(ctx, req); err != nil {
		return nil, err
	}
	return req, nil
}

func (s *Service) Deny(ctx context.Context, id, approverID string) (*domain.FileAccessRequest, error) {
	req, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	if req.Status != domain.FileAccessPending {
		return nil, domain.NewAppError("invalid_state", "request is not pending", 409)
	}
	now := time.Now()
	req.Status = domain.FileAccessDenied
	req.ApprovedBy = &approverID
	req.ResolvedAt = &now
	if err := s.repo.Update(ctx, req); err != nil {
		return nil, err
	}
	return req, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*domain.FileAccessRequest, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) ListByTenant(ctx context.Context, tenantID, status string) ([]*domain.FileAccessRequest, error) {
	return s.repo.ListByTenant(ctx, tenantID, status)
}

func (s *Service) SetResult(ctx context.Context, id, resultJSON string) error {
	req, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return domain.ErrNotFound
	}
	req.ResultJSON = resultJSON
	return s.repo.Update(ctx, req)
}
