package repository

import (
	"context"

	"github.com/zy-eagle/envnexus/services/platform-api/internal/domain"
)

type CommandTaskFilters struct {
	Status     string
	CreatedBy  string
	ApproverID string
	RiskLevel  string
}

type CommandTaskRepository interface {
	Create(ctx context.Context, task *domain.CommandTask) error
	GetByID(ctx context.Context, id string) (*domain.CommandTask, error)
	Update(ctx context.Context, task *domain.CommandTask) error
	ListByTenant(ctx context.Context, tenantID string, filters CommandTaskFilters, limit, offset int) ([]*domain.CommandTask, int64, error)
	ListPendingByApprover(ctx context.Context, tenantID, approverUserID string) ([]*domain.CommandTask, error)
	ListPendingByApproverRole(ctx context.Context, tenantID, roleID string) ([]*domain.CommandTask, error)
	CountPendingByApprover(ctx context.Context, tenantID, approverUserID string) (int64, error)
	ListExpired(ctx context.Context) ([]*domain.CommandTask, error)
}
