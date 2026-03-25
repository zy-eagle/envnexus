package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/domain"
	"gorm.io/gorm"
)

type PackageRepository interface {
	Create(ctx context.Context, pkg *domain.DownloadPackage) error
	GetByID(ctx context.Context, id string) (*domain.DownloadPackage, error)
	ListByTenant(ctx context.Context, tenantID string) ([]*domain.DownloadPackage, error)
}

type MySQLPackageRepository struct {
	db *gorm.DB
}

func NewMySQLPackageRepository(db *gorm.DB) *MySQLPackageRepository {
	return &MySQLPackageRepository{db: db}
}

func (r *MySQLPackageRepository) Create(ctx context.Context, pkg *domain.DownloadPackage) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(pkg).Error; err != nil {
			return err
		}

		// Enqueue build job
		payload := map[string]string{
			"package_id": pkg.ID,
			"tenant_id":  pkg.TenantID,
			"platform":   pkg.Platform,
			"arch":       pkg.Arch,
		}
		payloadBytes, _ := json.Marshal(payload)
		payloadStr := string(payloadBytes)

		job := map[string]interface{}{
			"id":           ulid.Make().String(),
			"tenant_id":    pkg.TenantID,
			"job_type":     "package_build",
			"status":       "queued",
			"payload_json": payloadStr,
			"priority":     5,
			"max_attempts": 3,
			"scheduled_at": time.Now(),
			"created_at":   time.Now(),
		}
		
		return tx.Table("jobs").Create(job).Error
	})
}

func (r *MySQLPackageRepository) GetByID(ctx context.Context, id string) (*domain.DownloadPackage, error) {
	var pkg domain.DownloadPackage
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&pkg).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &pkg, nil
}

func (r *MySQLPackageRepository) ListByTenant(ctx context.Context, tenantID string) ([]*domain.DownloadPackage, error) {
	var pkgs []*domain.DownloadPackage
	err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Find(&pkgs).Error
	return pkgs, err
}
