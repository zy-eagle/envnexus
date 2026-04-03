package domain

import "time"

type UserNotificationChannel struct {
	ID           string    `json:"id"            gorm:"primaryKey;size:26"`
	UserID       string    `json:"user_id"       gorm:"size:26;not null;index"`
	TenantID     string    `json:"tenant_id"     gorm:"size:26;not null;index"`
	ProviderID   string    `json:"provider_id"   gorm:"size:26;not null"`
	Provider     string    `json:"provider"      gorm:"size:32;not null"`
	ExternalID   string    `json:"external_id"   gorm:"size:255;not null"`
	ExternalName string    `json:"external_name" gorm:"size:128"`
	ChatID       string    `json:"chat_id"       gorm:"size:255"`
	Priority     int       `json:"priority"      gorm:"not null;default:0"`
	Verified     bool      `json:"verified"      gorm:"not null;default:false"`
	Status       string    `json:"status"        gorm:"size:32;not null;default:active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (u *UserNotificationChannel) TableName() string { return "user_notification_channels" }
