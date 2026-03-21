package repository

import (
	"context"

	"github.com/zy-eagle/envnexus/services/platform-api/internal/domain"
	"gorm.io/gorm"
)

type AuditRepository interface {
	Create(ctx context.Context, event *domain.AuditEvent) error
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
