package repository

import (
	"context"
	"errors"

	"github.com/zy-eagle/envnexus/services/platform-api/internal/domain"
	"gorm.io/gorm"
)

type SessionRepository interface {
	Create(ctx context.Context, session *domain.Session) error
	GetByID(ctx context.Context, id string) (*domain.Session, error)
	Update(ctx context.Context, session *domain.Session) error
	ListByTenant(ctx context.Context, tenantID string) ([]*domain.Session, error)
	ListByDevice(ctx context.Context, deviceID string) ([]*domain.Session, error)
}

type MySQLSessionRepository struct {
	db *gorm.DB
}

func NewMySQLSessionRepository(db *gorm.DB) *MySQLSessionRepository {
	return &MySQLSessionRepository{db: db}
}

func (r *MySQLSessionRepository) Create(ctx context.Context, session *domain.Session) error {
	return r.db.WithContext(ctx).Create(session).Error
}

func (r *MySQLSessionRepository) GetByID(ctx context.Context, id string) (*domain.Session, error) {
	var session domain.Session
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&session).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &session, nil
}

func (r *MySQLSessionRepository) Update(ctx context.Context, session *domain.Session) error {
	return r.db.WithContext(ctx).Save(session).Error
}

func (r *MySQLSessionRepository) ListByTenant(ctx context.Context, tenantID string) ([]*domain.Session, error) {
	var sessions []*domain.Session
	err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Order("started_at DESC").Find(&sessions).Error
	return sessions, err
}

func (r *MySQLSessionRepository) ListByDevice(ctx context.Context, deviceID string) ([]*domain.Session, error) {
	var sessions []*domain.Session
	err := r.db.WithContext(ctx).Where("device_id = ?", deviceID).Order("started_at DESC").Find(&sessions).Error
	return sessions, err
}
