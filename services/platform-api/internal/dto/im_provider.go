package dto

import "time"

type CreateIMProviderRequest struct {
	Provider   string `json:"provider" binding:"required,oneof=feishu wechat_work dingtalk slack"`
	Name       string `json:"name" binding:"required"`
	AppID      string `json:"app_id" binding:"required"`
	AppSecret  string `json:"app_secret" binding:"required"`
	WebhookURL string `json:"webhook_url"`
}

type UpdateIMProviderRequest struct {
	Name       *string `json:"name"`
	AppID      *string `json:"app_id"`
	AppSecret  *string `json:"app_secret"`
	WebhookURL *string `json:"webhook_url"`
	Status     *string `json:"status"`
}

type IMProviderResponse struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	Provider   string    `json:"provider"`
	Name       string    `json:"name"`
	AppID      string    `json:"app_id"`
	WebhookURL string    `json:"webhook_url"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type CreateNotificationChannelRequest struct {
	ProviderID   string `json:"provider_id" binding:"required"`
	ExternalID   string `json:"external_id" binding:"required"`
	ExternalName string `json:"external_name"`
	ChatID       string `json:"chat_id"`
}

type NotificationChannelResponse struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	TenantID     string    `json:"tenant_id"`
	ProviderID   string    `json:"provider_id"`
	Provider     string    `json:"provider"`
	ExternalID   string    `json:"external_id"`
	ExternalName string    `json:"external_name"`
	ChatID       string    `json:"chat_id"`
	Priority     int       `json:"priority"`
	Verified     bool      `json:"verified"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
