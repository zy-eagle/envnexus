package repository

import (
	"context"

	"github.com/zy-eagle/envnexus/services/platform-api/internal/domain"
	"gorm.io/gorm"
)

type SessionRepository interface {
	ListByTenant(ctx context.Context, tenantID string) ([]*domain.Session, error)
}

type MySQLSessionRepository struct {
	db *gorm.DB
}

func NewMySQLSessionRepository(db *gorm.DB) *MySQLSessionRepository {
	return &MySQLSessionRepository{db: db}
}

func (r *MySQLSessionRepository) ListByTenant(ctx context.Context, tenantID string) ([]*domain.Session, error) {
	var sessions []*domain.Session
	err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Order("started_at DESC").Find(&sessions).Error
	if err != nil {
		return nil, err
	}
	return sessions, nil
}
