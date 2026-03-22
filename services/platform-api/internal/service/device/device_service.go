package device

import (
	"context"
	"time"

	"github.com/zy-eagle/envnexus/services/platform-api/internal/domain"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/dto"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/repository"
)

type Service struct {
	deviceRepo repository.DeviceRepository
}

func NewService(deviceRepo repository.DeviceRepository) *Service {
	return &Service{
		deviceRepo: deviceRepo,
	}
}

func (s *Service) ListDevices(ctx context.Context, tenantID string) ([]*dto.DeviceResponse, error) {
	devices, err := s.deviceRepo.ListByTenantID(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	var resp []*dto.DeviceResponse
	for _, d := range devices {
		var lastSeen time.Time
		if d.LastHeartbeatAt != nil {
			lastSeen = *d.LastHeartbeatAt
		}
		
		resp = append(resp, &dto.DeviceResponse{
			ID:        d.ID,
			TenantID:  d.TenantID,
			Name:      d.Name,
			Hostname:  d.Hostname,
			OSType:    d.OSType,
			Status:    string(d.Status),
			LastSeen:  lastSeen,
			CreatedAt: d.CreatedAt,
			UpdatedAt: d.UpdatedAt,
		})
	}
	return resp, nil
}

func (s *Service) UpdateDevice(ctx context.Context, tenantID, id string, req dto.UpdateDeviceRequest) (*dto.DeviceResponse, error) {
	device, err := s.deviceRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if device == nil || device.TenantID != tenantID {
		return nil, context.Canceled // Or a proper not found error
	}

	if req.Name != "" {
		device.Name = req.Name
	}
	if req.Status != "" {
		device.Status = domain.DeviceStatus(req.Status)
	}

	if err := s.deviceRepo.Update(ctx, device); err != nil {
		return nil, err
	}

	var lastSeen time.Time
	if device.LastHeartbeatAt != nil {
		lastSeen = *device.LastHeartbeatAt
	}

	return &dto.DeviceResponse{
		ID:        device.ID,
		TenantID:  device.TenantID,
		Name:      device.Name,
		Hostname:  device.Hostname,
		OSType:    device.OSType,
		Status:    string(device.Status),
		LastSeen:  lastSeen,
		CreatedAt: device.CreatedAt,
		UpdatedAt: device.UpdatedAt,
	}, nil
}

func (s *Service) DeleteDevice(ctx context.Context, tenantID, id string) error {
	return s.deviceRepo.Delete(ctx, id, tenantID)
}
