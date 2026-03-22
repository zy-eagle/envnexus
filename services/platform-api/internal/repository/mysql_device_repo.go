package repository

import (
	"context"
	"errors"

	"github.com/zy-eagle/envnexus/services/platform-api/internal/domain"
	"gorm.io/gorm"
)

type EnrollmentRepository interface {
	Create(ctx context.Context, token *domain.EnrollmentToken) error
	GetByToken(ctx context.Context, token string) (*domain.EnrollmentToken, error)
	Update(ctx context.Context, token *domain.EnrollmentToken) error
}

type MySQLEnrollmentRepository struct {
	db *gorm.DB
}

func NewMySQLEnrollmentRepository(db *gorm.DB) *MySQLEnrollmentRepository {
	return &MySQLEnrollmentRepository{db: db}
}

func (r *MySQLEnrollmentRepository) Create(ctx context.Context, token *domain.EnrollmentToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

func (r *MySQLEnrollmentRepository) GetByToken(ctx context.Context, token string) (*domain.EnrollmentToken, error) {
	var et domain.EnrollmentToken
	err := r.db.WithContext(ctx).Where("token = ?", token).First(&et).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &et, nil
}

func (r *MySQLEnrollmentRepository) Update(ctx context.Context, token *domain.EnrollmentToken) error {
	return r.db.WithContext(ctx).Save(token).Error
}

type DeviceRepository interface {
	Create(ctx context.Context, device *domain.Device) error
	GetByID(ctx context.Context, id string) (*domain.Device, error)
	Update(ctx context.Context, device *domain.Device) error
	ListByTenantID(ctx context.Context, tenantID string) ([]*domain.Device, error)
	Delete(ctx context.Context, id string, tenantID string) error
}

type MySQLDeviceRepository struct {
	db *gorm.DB
}

func NewMySQLDeviceRepository(db *gorm.DB) *MySQLDeviceRepository {
	return &MySQLDeviceRepository{db: db}
}

func (r *MySQLDeviceRepository) Create(ctx context.Context, device *domain.Device) error {
	return r.db.WithContext(ctx).Create(device).Error
}

func (r *MySQLDeviceRepository) GetByID(ctx context.Context, id string) (*domain.Device, error) {
	var device domain.Device
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&device).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &device, nil
}

func (r *MySQLDeviceRepository) Update(ctx context.Context, device *domain.Device) error {
	return r.db.WithContext(ctx).Save(device).Error
}

func (r *MySQLDeviceRepository) ListByTenantID(ctx context.Context, tenantID string) ([]*domain.Device, error) {
	var devices []*domain.Device
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND deleted_at IS NULL", tenantID).Find(&devices).Error
	if err != nil {
		return nil, err
	}
	return devices, nil
}

func (r *MySQLDeviceRepository) Delete(ctx context.Context, id string, tenantID string) error {
	return r.db.WithContext(ctx).Model(&domain.Device{}).
		Where("id = ? AND tenant_id = ?", id, tenantID).
		Update("deleted_at", gorm.Expr("NOW()")).Error
}
