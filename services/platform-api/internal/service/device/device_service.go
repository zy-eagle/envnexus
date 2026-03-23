package device

import (
	"context"

	"github.com/zy-eagle/envnexus/services/platform-api/internal/domain"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/dto"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/repository"
)

type Service struct {
	deviceRepo repository.DeviceRepository
}

func NewService(deviceRepo repository.DeviceRepository) *Service {
	return &Service{deviceRepo: deviceRepo}
}

func (s *Service) ListDevices(ctx context.Context, tenantID string) ([]*dto.DeviceResponse, error) {
	devices, err := s.deviceRepo.ListByTenantID(ctx, tenantID)
	if err != nil {
		return nil, domain.ErrInternalError
	}

	var resp []*dto.DeviceResponse
	for _, d := range devices {
		hostname := ""
		if d.Hostname != nil {
			hostname = *d.Hostname
		}
		resp = append(resp, &dto.DeviceResponse{
			ID:              d.ID,
			TenantID:        d.TenantID,
			AgentProfileID:  d.AgentProfileID,
			DeviceName:      d.DeviceName,
			Hostname:        hostname,
			Platform:        d.Platform,
			Arch:            d.Arch,
			EnvironmentType: d.EnvironmentType,
			AgentVersion:    d.AgentVersion,
			Status:          string(d.Status),
			PolicyVersion:   d.PolicyVersion,
			LastSeenAt:      d.LastSeenAt,
			CreatedAt:       d.CreatedAt,
			UpdatedAt:       d.UpdatedAt,
		})
	}
	return resp, nil
}

func (s *Service) UpdateDevice(ctx context.Context, tenantID, id string, req dto.UpdateDeviceRequest) (*dto.DeviceResponse, error) {
	device, err := s.deviceRepo.GetByID(ctx, id)
	if err != nil {
		return nil, domain.ErrInternalError
	}
	if device == nil || device.TenantID != tenantID {
		return nil, domain.ErrDeviceNotFound
	}

	if req.DeviceName != "" {
		device.DeviceName = req.DeviceName
	}
	if req.Status != "" {
		device.Status = domain.DeviceStatus(req.Status)
	}

	if err := s.deviceRepo.Update(ctx, device); err != nil {
		return nil, domain.ErrInternalError
	}

	hostname := ""
	if device.Hostname != nil {
		hostname = *device.Hostname
	}
	return &dto.DeviceResponse{
		ID:              device.ID,
		TenantID:        device.TenantID,
		AgentProfileID:  device.AgentProfileID,
		DeviceName:      device.DeviceName,
		Hostname:        hostname,
		Platform:        device.Platform,
		Arch:            device.Arch,
		EnvironmentType: device.EnvironmentType,
		AgentVersion:    device.AgentVersion,
		Status:          string(device.Status),
		PolicyVersion:   device.PolicyVersion,
		LastSeenAt:      device.LastSeenAt,
		CreatedAt:       device.CreatedAt,
		UpdatedAt:       device.UpdatedAt,
	}, nil
}

func (s *Service) DeleteDevice(ctx context.Context, tenantID, id string) error {
	return s.deviceRepo.Delete(ctx, id, tenantID)
}
