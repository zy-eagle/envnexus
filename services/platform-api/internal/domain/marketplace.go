package domain

import (
	"context"
	"time"
)

// MarketplaceItemType classifies a marketplace entry.
type MarketplaceItemType string

const (
	MarketplaceItemTypeMcp      MarketplaceItemType = "mcp"
	MarketplaceItemTypeSkill    MarketplaceItemType = "skill"
	MarketplaceItemTypeSubagent MarketplaceItemType = "subagent"
	MarketplaceItemTypePlugin   MarketplaceItemType = "plugin"
	MarketplaceItemTypeRule     MarketplaceItemType = "rule"
)

// MarketplaceItemStatus is the publication state of a marketplace item.
type MarketplaceItemStatus string

const (
	MarketplaceItemStatusPublished MarketplaceItemStatus = "published"
	MarketplaceItemStatusDraft     MarketplaceItemStatus = "draft"
	MarketplaceItemStatusArchived  MarketplaceItemStatus = "archived"
)

// TenantSubscriptionStatus is the state of a tenant's subscription to an item.
type TenantSubscriptionStatus string

const (
	TenantSubscriptionStatusActive  TenantSubscriptionStatus = "active"
	TenantSubscriptionStatusRevoked TenantSubscriptionStatus = "revoked"
	TenantSubscriptionStatusPending TenantSubscriptionStatus = "pending"
)

// MarketplaceItem is a component offered in the marketplace (skill, MCP, subagent, plugin, or rule).
type MarketplaceItem struct {
	ID          string                `json:"id"            gorm:"primaryKey;size:26"`
	Type        MarketplaceItemType   `json:"type"          gorm:"column:type;size:32;not null;index"`
	Name        string                `json:"name"          gorm:"size:255;not null"`
	Description string                `json:"description"   gorm:"type:text"`
	Version     string                `json:"version"       gorm:"size:64;not null"`
	Author      string                `json:"author"        gorm:"size:255"`
	Payload     string                `json:"payload"       gorm:"type:json;not null"`
	Status      MarketplaceItemStatus `json:"status"        gorm:"size:32;not null;index;default:draft"`
	CreatedAt   time.Time             `json:"created_at"`
	UpdatedAt   time.Time             `json:"updated_at"`
}

// TableName returns the GORM table name for MarketplaceItem.
func (MarketplaceItem) TableName() string { return "marketplace_items" }

// TenantSubscription links a tenant to a subscribed marketplace item.
type TenantSubscription struct {
	ID        string                   `json:"id"         gorm:"primaryKey;size:26"`
	TenantID  string                   `json:"tenant_id"  gorm:"size:26;not null;uniqueIndex:ux_tenant_subscriptions_tenant_item"`
	ItemID    string                   `json:"item_id"    gorm:"size:26;not null;uniqueIndex:ux_tenant_subscriptions_tenant_item"`
	Status    TenantSubscriptionStatus `json:"status"     gorm:"size:32;not null;index;default:active"`
	CreatedAt time.Time                `json:"created_at"`
	UpdatedAt time.Time                `json:"updated_at"`
}

// TableName returns the GORM table name for TenantSubscription.
func (TenantSubscription) TableName() string { return "tenant_subscriptions" }

// MarketplaceRepository persists marketplace items and tenant subscriptions.
type MarketplaceRepository interface {
	CreateMarketplaceItem(ctx context.Context, item *MarketplaceItem) error
	GetMarketplaceItemByID(ctx context.Context, id string) (*MarketplaceItem, error)
	GetLatestMarketplaceItemByName(ctx context.Context, name string) (*MarketplaceItem, error)
	ListMarketplaceItems(ctx context.Context, itemType *MarketplaceItemType, status *MarketplaceItemStatus, page, pageSize int) ([]*MarketplaceItem, int64, error)
	UpdateMarketplaceItem(ctx context.Context, item *MarketplaceItem) error
	DeleteMarketplaceItem(ctx context.Context, id string) error

	CreateTenantSubscription(ctx context.Context, sub *TenantSubscription) error
	GetTenantSubscriptionByID(ctx context.Context, id string) (*TenantSubscription, error)
	GetTenantSubscriptionByTenantAndItem(ctx context.Context, tenantID, itemID string) (*TenantSubscription, error)
	ListTenantSubscriptionsByTenantID(ctx context.Context, tenantID string, page, pageSize int) ([]*TenantSubscription, int64, error)
	// ListActiveSubscribedItemsForTenant returns marketplace items the tenant has an active subscription to (join).
	ListActiveSubscribedItemsForTenant(ctx context.Context, tenantID string) ([]*MarketplaceItem, error)
	ListTenantSubscriptionsByItemID(ctx context.Context, itemID string, page, pageSize int) ([]*TenantSubscription, int64, error)
	UpdateTenantSubscription(ctx context.Context, sub *TenantSubscription) error
	DeleteTenantSubscription(ctx context.Context, id string) error
}
