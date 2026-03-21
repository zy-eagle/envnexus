package enrollment

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/domain"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/dto"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/repository"
)

type Service struct {
	enrollRepo repository.EnrollmentRepository
	deviceRepo repository.DeviceRepository
}

func NewService(enrollRepo repository.EnrollmentRepository, deviceRepo repository.DeviceRepository) *Service {
	return &Service{
		enrollRepo: enrollRepo,
		deviceRepo: deviceRepo,
	}
}

func (s *Service) CreateToken(ctx context.Context, tenantID string, req dto.CreateTokenRequest) (*dto.TokenResponse, error) {
	// Generate a secure random token string
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return nil, err
	}
	tokenStr := "enx_tok_" + hex.EncodeToString(bytes)

	now := time.Now()
	token := &domain.EnrollmentToken{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		Token:     tokenStr,
		MaxUses:   req.MaxUses,
		UsedCount: 0,
		ExpiresAt: now.Add(time.Duration(req.ExpiresIn) * time.Hour),
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.enrollRepo.Create(ctx, token); err != nil {
		return nil, err
	}

	return &dto.TokenResponse{
		ID:        token.ID,
		TenantID:  token.TenantID,
		Token:     token.Token,
		MaxUses:   token.MaxUses,
		UsedCount: token.UsedCount,
		ExpiresAt: token.ExpiresAt,
		CreatedAt: token.CreatedAt,
	}, nil
}

func (s *Service) EnrollAgent(ctx context.Context, req dto.AgentEnrollRequest) (*dto.AgentEnrollResponse, error) {
	// 1. Fetch and validate token
	token, err := s.enrollRepo.GetByToken(ctx, req.Token)
	if err != nil {
		return nil, err
	}
	if token == nil {
		return nil, errors.New("invalid enrollment token")
	}
	if !token.IsValid() {
		return nil, errors.New("enrollment token is expired or exhausted")
	}

	// 2. Check if device already exists
	existingDevice, err := s.deviceRepo.GetByID(ctx, req.DeviceID)
	if err != nil {
		return nil, err
	}

	if existingDevice == nil {
		// Create new device
		newDevice := &domain.Device{
			ID:        req.DeviceID,
			TenantID:  token.TenantID,
			Name:      req.Hostname, // Default name to hostname
			Hostname:  req.Hostname,
			OSType:    req.OSType,
			Status:    domain.DeviceStatusOnline,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := s.deviceRepo.Create(ctx, newDevice); err != nil {
			return nil, err
		}
	} else {
		// Update existing device
		existingDevice.Hostname = req.Hostname
		existingDevice.OSType = req.OSType
		existingDevice.RecordHeartbeat()
		if err := s.deviceRepo.Update(ctx, existingDevice); err != nil {
			return nil, err
		}
	}

	// 3. Mark token as used
	token.IncrementUsage()
	if err := s.enrollRepo.Update(ctx, token); err != nil {
		return nil, err
	}

	// 4. Generate device token (In a real app, sign a JWT or generate a secure random string)
	deviceToken := uuid.New().String()

	return &dto.AgentEnrollResponse{
		TenantID:    token.TenantID,
		DeviceToken: deviceToken,
	}, nil
}
