package auth

import (
	"context"
	"errors"

	"github.com/zy-eagle/envnexus/services/platform-api/internal/dto"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/repository"
)

type Service struct {
	userRepo repository.UserRepository
}

func NewService(userRepo repository.UserRepository) *Service {
	return &Service{
		userRepo: userRepo,
	}
}

func (s *Service) Login(ctx context.Context, req dto.LoginRequest) (*dto.LoginResponse, error) {
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("invalid email or password")
	}

	// For MVP: Plain text password check
	if user.PasswordHash != req.Password {
		return nil, errors.New("invalid email or password")
	}

	resp := &dto.LoginResponse{
		AccessToken: "mock-jwt-token-" + user.ID,
		ExpiresIn:   3600,
	}
	resp.User.ID = user.ID
	resp.User.TenantID = user.TenantID
	resp.User.Email = user.Email
	resp.User.DisplayName = user.DisplayName

	return resp, nil
}
