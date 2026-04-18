package repository

import (
	"context"

	"github.com/zy-eagle/envnexus/services/platform-api/internal/domain"
	"gorm.io/gorm"
)

type FileAccessRepository interface {
	Create(ctx context.Context, req *domain.FileAccessRequest) error
	GetByID(ctx context.Context, id string) (*domain.FileAccessRequest, error)
	ListByTenant(ctx context.Context, tenantID string, status string) ([]*domain.FileAccessRequest, error)
	Update(ctx context.Context, req *domain.FileAccessRequest) error
}

type MySQLFileAccessRepository struct {
	db *gorm.DB
}

func NewMySQLFileAccessRepository(db *gorm.DB) *MySQLFileAccessRepository {
	return &MySQLFileAccessRepository{db: db}
}

func (r *MySQLFileAccessRepository) Create(ctx context.Context, req *domain.FileAccessRequest) error {
	return r.db.WithContext(ctx).Create(req).Error
}

func (r *MySQLFileAccessRepository) GetByID(ctx context.Context, id string) (*domain.FileAccessRequest, error) {
	var req domain.FileAccessRequest
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&req).Error; err != nil {
		return nil, err
	}
	return &req, nil
}

func (r *MySQLFileAccessRepository) ListByTenant(ctx context.Context, tenantID string, status string) ([]*domain.FileAccessRequest, error) {
	q := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var requests []*domain.FileAccessRequest
	if err := q.Order("created_at DESC").Limit(200).Find(&requests).Error; err != nil {
		return nil, err
	}
	return requests, nil
}

func (r *MySQLFileAccessRepository) Update(ctx context.Context, req *domain.FileAccessRequest) error {
	return r.db.WithContext(ctx).Save(req).Error
}
