package repository

import (
	"context"
	"errors"

	"github.com/zy-eagle/envnexus/services/platform-api/internal/domain"
	"gorm.io/gorm"
)

type RoleRepository interface {
	Create(ctx context.Context, role *domain.Role) error
	GetByID(ctx context.Context, id string) (*domain.Role, error)
	ListByTenant(ctx context.Context, tenantID string) ([]*domain.Role, error)
	Update(ctx context.Context, role *domain.Role) error
	Delete(ctx context.Context, id string) error
}

type MySQLRoleRepository struct {
	db *gorm.DB
}

func NewMySQLRoleRepository(db *gorm.DB) *MySQLRoleRepository {
	return &MySQLRoleRepository{db: db}
}

func (r *MySQLRoleRepository) Create(ctx context.Context, role *domain.Role) error {
	return r.db.WithContext(ctx).Create(role).Error
}

func (r *MySQLRoleRepository) GetByID(ctx context.Context, id string) (*domain.Role, error) {
	var role domain.Role
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&role).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}

func (r *MySQLRoleRepository) ListByTenant(ctx context.Context, tenantID string) ([]*domain.Role, error) {
	var roles []*domain.Role
	err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Find(&roles).Error
	return roles, err
}

func (r *MySQLRoleRepository) Update(ctx context.Context, role *domain.Role) error {
	return r.db.WithContext(ctx).Save(role).Error
}

func (r *MySQLRoleRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&domain.Role{}, "id = ?", id).Error
}
