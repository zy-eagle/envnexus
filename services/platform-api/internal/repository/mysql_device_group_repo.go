package repository

import (
	"context"

	"github.com/zy-eagle/envnexus/services/platform-api/internal/domain"
	"gorm.io/gorm"
)

type DeviceGroupRepository interface {
	CreateGroup(ctx context.Context, g *domain.DeviceGroup) error
	GetGroup(ctx context.Context, id string) (*domain.DeviceGroup, error)
	ListGroups(ctx context.Context, tenantID string) ([]*domain.DeviceGroup, error)
	UpdateGroup(ctx context.Context, g *domain.DeviceGroup) error
	DeleteGroup(ctx context.Context, id string) error

	AddMembers(ctx context.Context, members []*domain.DeviceGroupMember) error
	RemoveMember(ctx context.Context, groupID, deviceID string) error
	ListMembers(ctx context.Context, groupID string) ([]*domain.DeviceGroupMember, error)

	CreateBatchTask(ctx context.Context, t *domain.BatchTask) error
	GetBatchTask(ctx context.Context, id string) (*domain.BatchTask, error)
	ListBatchTasks(ctx context.Context, tenantID string) ([]*domain.BatchTask, error)
	UpdateBatchTask(ctx context.Context, t *domain.BatchTask) error
}

type MySQLDeviceGroupRepository struct {
	db *gorm.DB
}

func NewMySQLDeviceGroupRepository(db *gorm.DB) *MySQLDeviceGroupRepository {
	return &MySQLDeviceGroupRepository{db: db}
}

func (r *MySQLDeviceGroupRepository) CreateGroup(ctx context.Context, g *domain.DeviceGroup) error {
	return r.db.WithContext(ctx).Create(g).Error
}

func (r *MySQLDeviceGroupRepository) GetGroup(ctx context.Context, id string) (*domain.DeviceGroup, error) {
	var g domain.DeviceGroup
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&g).Error; err != nil {
		return nil, err
	}
	return &g, nil
}

func (r *MySQLDeviceGroupRepository) ListGroups(ctx context.Context, tenantID string) ([]*domain.DeviceGroup, error) {
	var groups []*domain.DeviceGroup
	if err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Order("created_at DESC").Find(&groups).Error; err != nil {
		return nil, err
	}
	return groups, nil
}

func (r *MySQLDeviceGroupRepository) UpdateGroup(ctx context.Context, g *domain.DeviceGroup) error {
	return r.db.WithContext(ctx).Save(g).Error
}

func (r *MySQLDeviceGroupRepository) DeleteGroup(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("device_group_id = ?", id).Delete(&domain.DeviceGroupMember{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ?", id).Delete(&domain.DeviceGroup{}).Error
	})
}

func (r *MySQLDeviceGroupRepository) AddMembers(ctx context.Context, members []*domain.DeviceGroupMember) error {
	if len(members) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&members).Error
}

func (r *MySQLDeviceGroupRepository) RemoveMember(ctx context.Context, groupID, deviceID string) error {
	return r.db.WithContext(ctx).Where("device_group_id = ? AND device_id = ?", groupID, deviceID).Delete(&domain.DeviceGroupMember{}).Error
}

func (r *MySQLDeviceGroupRepository) ListMembers(ctx context.Context, groupID string) ([]*domain.DeviceGroupMember, error) {
	var members []*domain.DeviceGroupMember
	if err := r.db.WithContext(ctx).Where("device_group_id = ?", groupID).Find(&members).Error; err != nil {
		return nil, err
	}
	return members, nil
}

func (r *MySQLDeviceGroupRepository) CreateBatchTask(ctx context.Context, t *domain.BatchTask) error {
	return r.db.WithContext(ctx).Create(t).Error
}

func (r *MySQLDeviceGroupRepository) GetBatchTask(ctx context.Context, id string) (*domain.BatchTask, error) {
	var t domain.BatchTask
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *MySQLDeviceGroupRepository) ListBatchTasks(ctx context.Context, tenantID string) ([]*domain.BatchTask, error) {
	var tasks []*domain.BatchTask
	if err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Order("created_at DESC").Limit(200).Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

func (r *MySQLDeviceGroupRepository) UpdateBatchTask(ctx context.Context, t *domain.BatchTask) error {
	return r.db.WithContext(ctx).Save(t).Error
}
