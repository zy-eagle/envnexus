package repository

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// UsageMetricRow mirrors the usage_metrics table.
type UsageMetricRow struct {
	ID          string    `gorm:"primaryKey;size:26"`
	TenantID    string    `gorm:"size:26;not null"`
	MetricType  string    `gorm:"size:64;not null"`
	Value       int64     `gorm:"not null;default:0"`
	PeriodStart time.Time `gorm:"not null"`
	PeriodEnd   time.Time `gorm:"not null"`
	CreatedAt   time.Time
}

func (UsageMetricRow) TableName() string { return "usage_metrics" }

// MetricsRepository abstracts usage metric persistence.
type MetricsRepository interface {
	// Upsert increments a metric counter for the given period using an atomic upsert.
	Upsert(ctx context.Context, tenantID, metricType string, delta int64, periodStart, periodEnd time.Time) error
	// ListByTenantAndPeriod returns all metric rows for a tenant from a given start time.
	ListByTenantAndPeriod(ctx context.Context, tenantID string, from time.Time) ([]UsageMetricRow, error)
}

type MySQLMetricsRepository struct {
	db *gorm.DB
}

func NewMySQLMetricsRepository(db *gorm.DB) *MySQLMetricsRepository {
	return &MySQLMetricsRepository{db: db}
}

func (r *MySQLMetricsRepository) Upsert(ctx context.Context, tenantID, metricType string, delta int64, periodStart, periodEnd time.Time) error {
	return r.db.WithContext(ctx).
		Exec(`INSERT INTO usage_metrics (id, tenant_id, metric_type, value, period_start, period_end, created_at)
			  VALUES (UUID(), ?, ?, ?, ?, ?, NOW())
			  ON DUPLICATE KEY UPDATE value = value + ?`,
			tenantID, metricType, delta, periodStart, periodEnd, delta,
		).Error
}

func (r *MySQLMetricsRepository) ListByTenantAndPeriod(ctx context.Context, tenantID string, from time.Time) ([]UsageMetricRow, error) {
	var rows []UsageMetricRow
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND period_start >= ?", tenantID, from).
		Order("period_start ASC").
		Find(&rows).Error
	return rows, err
}
