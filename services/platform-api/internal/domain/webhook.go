package domain

import (
	"encoding/json"
	"time"
)

type WebhookSubscription struct {
	ID           string    `json:"id"            gorm:"primaryKey;size:26"`
	TenantID     string    `json:"tenant_id"     gorm:"size:26;not null;index"`
	Name         string    `json:"name"          gorm:"size:128;not null"`
	URL          string    `json:"url"           gorm:"size:2048;not null"`
	Secret       string    `json:"-"             gorm:"size:255;not null"`
	EventTypesJSON string  `json:"-"             gorm:"type:json;not null"`
	Status       string    `json:"status"        gorm:"size:32;not null;default:active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (w *WebhookSubscription) TableName() string { return "webhook_subscriptions" }

func (w *WebhookSubscription) EventTypes() []string {
	var types []string
	if err := json.Unmarshal([]byte(w.EventTypesJSON), &types); err != nil {
		return nil
	}
	return types
}

func (w *WebhookSubscription) ListensTo(eventType string) bool {
	for _, t := range w.EventTypes() {
		if t == "*" || t == eventType {
			return true
		}
	}
	return false
}

type WebhookDelivery struct {
	ID               string     `json:"id"                gorm:"primaryKey;size:26"`
	SubscriptionID   string     `json:"subscription_id"   gorm:"size:26;not null;index"`
	TenantID         string     `json:"tenant_id"         gorm:"size:26;not null"`
	EventType        string     `json:"event_type"        gorm:"size:64;not null"`
	PayloadJSON      string     `json:"-"                 gorm:"type:json;not null"`
	IdempotencyKey   string     `json:"idempotency_key"   gorm:"size:128"`
	Status           string     `json:"status"            gorm:"size:32;not null;default:pending"`
	HTTPStatus       *int       `json:"http_status"       gorm:"column:http_status"`
	ResponseBody     *string    `json:"response_body,omitempty" gorm:"type:text"`
	AttemptCount     int        `json:"attempt_count"     gorm:"not null;default:0"`
	NextRetryAt      *time.Time `json:"next_retry_at"     gorm:"column:next_retry_at"`
	DeliveredAt      *time.Time `json:"delivered_at"`
	CreatedAt        time.Time  `json:"created_at"`
}

func (w *WebhookDelivery) TableName() string { return "webhook_deliveries" }
