package session

import (
	"context"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/zy-eagle/envnexus/services/platform-api/internal/domain"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/dto"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/repository"
)

type Service struct {
	sessionRepo  repository.SessionRepository
	approvalRepo repository.ApprovalRequestRepository
	deviceRepo   repository.DeviceRepository
	auditRepo    repository.AuditRepository
}

func NewService(
	sessionRepo repository.SessionRepository,
	approvalRepo repository.ApprovalRequestRepository,
	deviceRepo repository.DeviceRepository,
	auditRepo repository.AuditRepository,
) *Service {
	return &Service{
		sessionRepo:  sessionRepo,
		approvalRepo: approvalRepo,
		deviceRepo:   deviceRepo,
		auditRepo:    auditRepo,
	}
}

func (s *Service) ListByTenant(ctx context.Context, tenantID string) ([]*domain.Session, error) {
	return s.sessionRepo.ListByTenant(ctx, tenantID)
}

func (s *Service) CreateSession(ctx context.Context, req dto.CreateSessionRequest) (*domain.Session, error) {
	device, err := s.deviceRepo.GetByID(ctx, req.DeviceID)
	if err != nil {
		return nil, domain.ErrInternalError
	}
	if device == nil {
		return nil, domain.ErrDeviceNotFound
	}
	if device.IsRevoked() {
		return nil, domain.ErrDeviceRevoked
	}

	now := time.Now()
	session := &domain.Session{
		ID:            ulid.Make().String(),
		TenantID:      device.TenantID,
		DeviceID:      req.DeviceID,
		Transport:     req.Transport,
		Status:        domain.SessionStatusCreated,
		InitiatorType: req.InitiatorType,
		StartedAt:     now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, domain.ErrInternalError
	}

	return session, nil
}

func (s *Service) GetByID(ctx context.Context, sessionID string) (*domain.Session, error) {
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, domain.ErrInternalError
	}
	if session == nil {
		return nil, domain.ErrSessionNotFound
	}
	return session, nil
}

func (s *Service) ApproveSession(ctx context.Context, sessionID string, approvalRequestID string, approverUserID string, comment string) error {
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil || session == nil {
		return domain.ErrSessionNotFound
	}

	approval, err := s.approvalRepo.GetByID(ctx, approvalRequestID)
	if err != nil || approval == nil {
		return domain.ErrApprovalNotFound
	}
	if approval.SessionID != sessionID {
		return domain.ErrApprovalInvalidState
	}
	if approval.IsExpired() {
		approval.Status = domain.ApprovalStatusExpired
		_ = s.approvalRepo.Update(ctx, approval)
		return domain.ErrApprovalExpired
	}
	if !approval.CanApprove() {
		return domain.ErrApprovalInvalidState
	}

	now := time.Now()
	approval.Status = domain.ApprovalStatusApproved
	approval.ApproverUserID = &approverUserID
	approval.ApprovedAt = &now
	approval.UpdatedAt = now

	return s.approvalRepo.Update(ctx, approval)
}

func (s *Service) DenySession(ctx context.Context, sessionID string, approvalRequestID string, reason string) error {
	approval, err := s.approvalRepo.GetByID(ctx, approvalRequestID)
	if err != nil || approval == nil {
		return domain.ErrApprovalNotFound
	}
	if approval.SessionID != sessionID {
		return domain.ErrApprovalInvalidState
	}
	if !approval.CanDeny() {
		return domain.ErrApprovalInvalidState
	}

	approval.Status = domain.ApprovalStatusDenied
	approval.UpdatedAt = time.Now()
	return s.approvalRepo.Update(ctx, approval)
}

func (s *Service) AbortSession(ctx context.Context, sessionID string, reason string) error {
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil || session == nil {
		return domain.ErrSessionNotFound
	}

	if !session.TransitionTo(domain.SessionStatusAborted) {
		return domain.ErrSessionInvalidState
	}
	return s.sessionRepo.Update(ctx, session)
}
