package metrics

import (
	"context"
	"log/slog"
	"time"

	"github.com/oklog/ulid/v2"
	"gorm.io/gorm"
)

// MetricType enumerates the supported metric counters.
const (
	MetricDeviceRegistrations = "device_registrations"
	MetricSessions            = "sessions_created"
	MetricLLMCalls            = "llm_calls"
	MetricAuditEvents         = "audit_events"
	MetricApprovals           = "approvals_total"
	MetricWebhookDeliveries   = "webhook_deliveries"
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

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// Increment increments a metric counter for the current calendar month.
func (s *Service) Increment(ctx context.Context, tenantID, metricType string, delta int64) {
	now := time.Now().UTC()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.AddDate(0, 1, 0).Add(-time.Nanosecond)

	err := s.db.WithContext(ctx).
		Exec(`INSERT INTO usage_metrics (id, tenant_id, metric_type, value, period_start, period_end, created_at)
			  VALUES (?, ?, ?, ?, ?, ?, ?)
			  ON DUPLICATE KEY UPDATE value = value + ?`,
			ulid.Make().String(), tenantID, metricType, delta, periodStart, periodEnd, now, delta,
		).Error
	if err != nil {
		slog.Warn("Failed to increment metric", "tenant_id", tenantID, "metric_type", metricType, "error", err)
	}
}

// GetCurrentPeriod returns usage for the current month.
func (s *Service) GetCurrentPeriod(ctx context.Context, tenantID string) (map[string]int64, error) {
	now := time.Now().UTC()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	var rows []UsageMetricRow
	err := s.db.WithContext(ctx).
		Where("tenant_id = ? AND period_start = ?", tenantID, periodStart).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make(map[string]int64)
	for _, r := range rows {
		result[r.MetricType] = r.Value
	}
	return result, nil
}

// GetHistory returns monthly usage for the past N months.
func (s *Service) GetHistory(ctx context.Context, tenantID string, months int) ([]map[string]interface{}, error) {
	if months < 1 || months > 24 {
		months = 6
	}
	start := time.Now().UTC().AddDate(0, -months, 0)

	var rows []UsageMetricRow
	err := s.db.WithContext(ctx).
		Where("tenant_id = ? AND period_start >= ?", tenantID, start).
		Order("period_start ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	// Group by period
	byPeriod := make(map[time.Time]map[string]int64)
	for _, r := range rows {
		if _, ok := byPeriod[r.PeriodStart]; !ok {
			byPeriod[r.PeriodStart] = make(map[string]int64)
		}
		byPeriod[r.PeriodStart][r.MetricType] = r.Value
	}

	var result []map[string]interface{}
	for period, metrics := range byPeriod {
		entry := map[string]interface{}{
			"period": period.Format("2006-01"),
		}
		for k, v := range metrics {
			entry[k] = v
		}
		result = append(result, entry)
	}
	return result, nil
}
