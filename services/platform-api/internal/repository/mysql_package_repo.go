package repository

import (
	"context"
	"errors"

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
	return r.db.WithContext(ctx).Create(pkg).Error
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
