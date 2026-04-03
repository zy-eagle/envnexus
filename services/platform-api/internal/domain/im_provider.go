package domain

import "time"

type IMProviderType string

const (
	IMProviderFeishu     IMProviderType = "feishu"
	IMProviderWeChatWork IMProviderType = "wechat_work"
	IMProviderDingTalk   IMProviderType = "dingtalk"
	IMProviderSlack      IMProviderType = "slack"
)

type IMProvider struct {
	ID         string         `json:"id"          gorm:"primaryKey;size:26"`
	TenantID   string         `json:"tenant_id"   gorm:"size:26;not null;index"`
	Provider   IMProviderType `json:"provider"    gorm:"size:32;not null"`
	Name       string         `json:"name"        gorm:"size:128;not null"`
	ConfigJSON string         `json:"-"           gorm:"type:text;not null"`
	WebhookURL string         `json:"webhook_url" gorm:"size:512"`
	Status     string         `json:"status"      gorm:"size:32;not null;default:active"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

func (i *IMProvider) TableName() string { return "im_providers" }
