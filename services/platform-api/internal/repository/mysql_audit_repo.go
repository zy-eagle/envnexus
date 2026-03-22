package repository

import (
	"context"

	"github.com/zy-eagle/envnexus/services/platform-api/internal/domain"
	"gorm.io/gorm"
)

type AuditRepository interface {
	Create(ctx context.Context, event *domain.AuditEvent) error
	ListByTenant(ctx context.Context, tenantID string) ([]*domain.AuditEvent, error)
}

type MySQLAuditRepository struct {
	db *gorm.DB
}

func NewMySQLAuditRepository(db *gorm.DB) *MySQLAuditRepository {
	return &MySQLAuditRepository{db: db}
}

func (r *MySQLAuditRepository) Create(ctx context.Context, event *domain.AuditEvent) error {
	return r.db.WithContext(ctx).Create(event).Error
}

func (r *MySQLAuditRepository) ListByTenant(ctx context.Context, tenantID string) ([]*domain.AuditEvent, error) {
	var events []*domain.AuditEvent
	err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Order("created_at DESC").Find(&events).Error
	if err != nil {
		return nil, err
	}
	return events, nil
}
