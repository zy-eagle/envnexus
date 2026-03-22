package repository

import (
	"context"

	"github.com/zy-eagle/envnexus/services/platform-api/internal/domain"
	"gorm.io/gorm"
)

type ModelProfileRepository interface {
	ListByTenant(ctx context.Context, tenantID string) ([]*domain.ModelProfile, error)
	Create(ctx context.Context, profile *domain.ModelProfile) error
}

type MySQLModelProfileRepository struct {
	db *gorm.DB
}

func NewMySQLModelProfileRepository(db *gorm.DB) *MySQLModelProfileRepository {
	return &MySQLModelProfileRepository{db: db}
}

func (r *MySQLModelProfileRepository) ListByTenant(ctx context.Context, tenantID string) ([]*domain.ModelProfile, error) {
	var profiles []*domain.ModelProfile
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND deleted_at IS NULL", tenantID).Find(&profiles).Error
	if err != nil {
		return nil, err
	}
	return profiles, nil
}

func (r *MySQLModelProfileRepository) Create(ctx context.Context, profile *domain.ModelProfile) error {
	return r.db.WithContext(ctx).Create(profile).Error
}
