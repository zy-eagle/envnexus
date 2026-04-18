package file_access

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/zy-eagle/envnexus/services/platform-api/internal/domain"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/infrastructure"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/repository"
)

type Service struct {
	repo          repository.FileAccessRepository
	auditRepo     repository.AuditRepository
	gatewayClient *infrastructure.GatewayClient
	minioClient   *infrastructure.MinIOClient
}

func NewService(repo repository.FileAccessRepository, auditRepo repository.AuditRepository, gatewayClient *infrastructure.GatewayClient, minioClient *infrastructure.MinIOClient) *Service {
	return &Service{repo: repo, auditRepo: auditRepo, gatewayClient: gatewayClient, minioClient: minioClient}
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

	go s.dispatchToDevice(context.Background(), req)

	return req, nil
}

func (s *Service) dispatchToDevice(ctx context.Context, req *domain.FileAccessRequest) {
	if s.gatewayClient == nil {
		slog.Warn("[file_access] gateway client not configured, cannot dispatch")
		return
	}

	payload := map[string]interface{}{
		"request_id": req.ID,
		"path":       req.Path,
		"action":     string(req.Action),
	}

	if req.Action == domain.FileAccessDownload && s.minioClient != nil {
		objectKey := fmt.Sprintf("file-access/%s/%s", req.TenantID, req.ID)
		uploadURL, err := s.minioClient.PresignedPutURL(ctx, objectKey, 30*time.Minute)
		if err != nil {
			slog.Error("[file_access] Failed to generate presigned upload URL",
				"request_id", req.ID,
				"error", err,
			)
			return
		}
		downloadURL, err := s.minioClient.PresignedGetURL(ctx, objectKey, 24*time.Hour)
		if err != nil {
			slog.Error("[file_access] Failed to generate presigned download URL",
				"request_id", req.ID,
				"error", err,
			)
			return
		}
		payload["upload_url"] = uploadURL.String()
		payload["download_url"] = downloadURL.String()
	}

	evt := infrastructure.SessionEvent{
		EventID:   fmt.Sprintf("fa_%s", req.ID),
		EventType: "file.execute",
		TenantID:  req.TenantID,
		DeviceID:  req.DeviceID,
		SessionID: req.ID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Payload:   payload,
	}

	if err := s.gatewayClient.SendToDevice(ctx, req.DeviceID, evt); err != nil {
		slog.Error("[file_access] Failed to dispatch to device",
			"request_id", req.ID,
			"device_id", req.DeviceID,
			"error", err,
		)
	}
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
