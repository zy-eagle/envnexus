package repository

import (
	"context"

	"github.com/zy-eagle/envnexus/services/platform-api/internal/domain"
	"gorm.io/gorm"
)

type AuditRepository interface {
	Create(ctx context.Context, event *domain.AuditEvent) error
	CreateBatch(ctx context.Context, events []*domain.AuditEvent) error
	ListByTenant(ctx context.Context, tenantID string, filters AuditFilters) ([]*domain.AuditEvent, error)
}

type AuditFilters struct {
	DeviceID     string
	SessionID    string
	EventType    string
	StartAt      string
	EndAt        string
	IncludeArchived bool
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

func (r *MySQLAuditRepository) CreateBatch(ctx context.Context, events []*domain.AuditEvent) error {
	if len(events) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&events).Error
}

func (r *MySQLAuditRepository) ListByTenant(ctx context.Context, tenantID string, filters AuditFilters) ([]*domain.AuditEvent, error) {
	query := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID)
	if !filters.IncludeArchived {
		query = query.Where("archived = ?", false)
	}
	if filters.DeviceID != "" {
		query = query.Where("device_id = ?", filters.DeviceID)
	}
	if filters.SessionID != "" {
		query = query.Where("session_id = ?", filters.SessionID)
	}
	if filters.EventType != "" {
		query = query.Where("event_type = ?", filters.EventType)
	}
	if filters.StartAt != "" {
		query = query.Where("created_at >= ?", filters.StartAt)
	}
	if filters.EndAt != "" {
		query = query.Where("created_at <= ?", filters.EndAt)
	}

	var events []*domain.AuditEvent
	err := query.Order("created_at DESC").Limit(200).Find(&events).Error
	return events, err
}
