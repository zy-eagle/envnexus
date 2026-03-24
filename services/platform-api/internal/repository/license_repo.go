package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

// LicenseRow mirrors the licenses table.
type LicenseRow struct {
	ID           string     `gorm:"primaryKey;size:26"`
	TenantID     string     `gorm:"size:26;not null;index"`
	LicenseKey   string     `gorm:"size:512;not null;uniqueIndex"`
	PlanCode     string     `gorm:"size:32;not null;default:enterprise"`
	MaxDevices   int        `gorm:"not null;default:100"`
	FeaturesJSON *string    `gorm:"type:json"`
	IssuedAt     time.Time
	ExpiresAt    time.Time
	Status       string     `gorm:"size:32;not null;default:active"`
	CreatedAt    time.Time
}

func (LicenseRow) TableName() string { return "licenses" }

// LicenseRepository abstracts license persistence.
type LicenseRepository interface {
	Create(ctx context.Context, row *LicenseRow) error
	GetActiveByTenant(ctx context.Context, tenantID string) (*LicenseRow, error)
	GetLatestByTenant(ctx context.Context, tenantID string) (*LicenseRow, error)
	Revoke(ctx context.Context, licenseID string) error
}

type MySQLLicenseRepository struct {
	db *gorm.DB
}

func NewMySQLLicenseRepository(db *gorm.DB) *MySQLLicenseRepository {
	return &MySQLLicenseRepository{db: db}
}

func (r *MySQLLicenseRepository) Create(ctx context.Context, row *LicenseRow) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *MySQLLicenseRepository) GetActiveByTenant(ctx context.Context, tenantID string) (*LicenseRow, error) {
	var row LicenseRow
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND status = ? AND expires_at > ?", tenantID, "active", time.Now()).
		Order("expires_at DESC").
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

func (r *MySQLLicenseRepository) GetLatestByTenant(ctx context.Context, tenantID string) (*LicenseRow, error) {
	var row LicenseRow
	err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("expires_at DESC").
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

func (r *MySQLLicenseRepository) Revoke(ctx context.Context, licenseID string) error {
	return r.db.WithContext(ctx).
		Table("licenses").
		Where("id = ?", licenseID).
		Updates(map[string]interface{}{"status": "revoked", "updated_at": time.Now()}).Error
}
