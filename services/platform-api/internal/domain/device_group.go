package domain

import "time"

type DeviceGroup struct {
	ID          string    `json:"id"          gorm:"primaryKey;size:26"`
	TenantID    string    `json:"tenant_id"   gorm:"size:26;not null;index"`
	Name        string    `json:"name"        gorm:"size:255;not null"`
	Description string    `json:"description" gorm:"type:text"`
	FilterJSON  string    `json:"filter"      gorm:"column:filter_json;type:text"`
	CreatedBy   string    `json:"created_by"  gorm:"size:26;not null"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (DeviceGroup) TableName() string { return "device_groups" }

type DeviceGroupMember struct {
	ID            string    `json:"id"             gorm:"primaryKey;size:26"`
	DeviceGroupID string    `json:"device_group_id" gorm:"size:26;not null;index"`
	DeviceID      string    `json:"device_id"       gorm:"size:26;not null;index"`
	CreatedAt     time.Time `json:"created_at"`
}

func (DeviceGroupMember) TableName() string { return "device_group_members" }

type BatchTask struct {
	ID            string    `json:"id"             gorm:"primaryKey;size:26"`
	TenantID      string    `json:"tenant_id"      gorm:"size:26;not null;index"`
	DeviceGroupID string    `json:"device_group_id" gorm:"size:26;not null;index"`
	CommandTaskID string    `json:"command_task_id"  gorm:"size:26;not null"`
	Strategy      string    `json:"strategy"        gorm:"size:32;not null;default:'all_at_once'"`
	BatchSize     int       `json:"batch_size"      gorm:"not null;default:0"`
	TotalDevices  int       `json:"total_devices"   gorm:"not null"`
	Completed     int       `json:"completed"       gorm:"not null;default:0"`
	Failed        int       `json:"failed"          gorm:"not null;default:0"`
	Status        string    `json:"status"          gorm:"size:32;not null;default:'pending'"`
	CreatedBy     string    `json:"created_by"      gorm:"size:26;not null"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (BatchTask) TableName() string { return "batch_tasks" }
