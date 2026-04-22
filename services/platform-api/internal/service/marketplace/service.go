package marketplace

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
	"gorm.io/gorm"

	"github.com/zy-eagle/envnexus/services/platform-api/internal/domain"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/dto"
)

type Service struct {
	repo domain.MarketplaceRepository
}

func NewService(repo domain.MarketplaceRepository) *Service {
	return &Service{repo: repo}
}

// ListItems lists marketplace items with optional type/status filters.
func (s *Service) ListItems(
	ctx context.Context, itemType *domain.MarketplaceItemType, status *domain.MarketplaceItemStatus, page, pageSize int,
) ([]*dto.MarketplaceItemResponse, int64, error) {
	items, total, err := s.repo.ListMarketplaceItems(ctx, itemType, status, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	out := make([]*dto.MarketplaceItemResponse, 0, len(items))
	for _, it := range items {
		out = append(out, itemToDTO(it))
	}
	return out, total, nil
}

func itemToDTO(it *domain.MarketplaceItem) *dto.MarketplaceItemResponse {
	if it == nil {
		return nil
	}
	return &dto.MarketplaceItemResponse{
		ID:          it.ID,
		Type:        it.Type,
		Name:        it.Name,
		Description: it.Description,
		Version:     it.Version,
		Author:      it.Author,
		Payload:     it.Payload,
		Status:      it.Status,
		CreatedAt:   it.CreatedAt,
		UpdatedAt:   it.UpdatedAt,
	}
}

// Subscribe adds or reactivates a tenant subscription to a published item.
func (s *Service) Subscribe(ctx context.Context, tenantID, itemID string) (*dto.TenantSubscriptionResponse, error) {
	item, err := s.repo.GetMarketplaceItemByID(ctx, itemID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrMarketplaceItemNotFound
		}
		return nil, err
	}
	if item.Status != domain.MarketplaceItemStatusPublished {
		return nil, domain.ErrMarketplaceItemNotPublished
	}
	existing, err := s.repo.GetTenantSubscriptionByTenantAndItem(ctx, tenantID, itemID)
	if err == nil && existing != nil {
		switch existing.Status {
		case domain.TenantSubscriptionStatusActive:
			return nil, domain.ErrMarketplaceAlreadySubscribed
		case domain.TenantSubscriptionStatusRevoked, domain.TenantSubscriptionStatusPending:
			now := time.Now()
			existing.Status = domain.TenantSubscriptionStatusActive
			existing.UpdatedAt = now
			if err := s.repo.UpdateTenantSubscription(ctx, existing); err != nil {
				return nil, err
			}
			return subToDTO(existing), nil
		default:
			return nil, domain.ErrInvalidRequest
		}
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	now := time.Now()
	sub := &domain.TenantSubscription{
		ID:        ulid.Make().String(),
		TenantID:  tenantID,
		ItemID:    itemID,
		Status:    domain.TenantSubscriptionStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.CreateTenantSubscription(ctx, sub); err != nil {
		return nil, err
	}
	return subToDTO(sub), nil
}

// Unsubscribe marks a subscription as revoked. Idempotent if already revoked.
func (s *Service) Unsubscribe(ctx context.Context, tenantID, itemID string) error {
	sub, err := s.repo.GetTenantSubscriptionByTenantAndItem(ctx, tenantID, itemID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrMarketplaceSubscriptionNotFound
		}
		return err
	}
	if sub.Status == domain.TenantSubscriptionStatusRevoked {
		return nil
	}
	now := time.Now()
	sub.Status = domain.TenantSubscriptionStatusRevoked
	sub.UpdatedAt = now
	return s.repo.UpdateTenantSubscription(ctx, sub)
}

// ListSubscriptions returns paginated tenant subscriptions.
func (s *Service) ListSubscriptions(ctx context.Context, tenantID string, page, pageSize int) ([]*dto.TenantSubscriptionResponse, int64, error) {
	subs, total, err := s.repo.ListTenantSubscriptionsByTenantID(ctx, tenantID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	out := make([]*dto.TenantSubscriptionResponse, 0, len(subs))
	for _, sub := range subs {
		out = append(out, subToDTO(sub))
	}
	return out, total, nil
}

func subToDTO(s *domain.TenantSubscription) *dto.TenantSubscriptionResponse {
	if s == nil {
		return nil
	}
	return &dto.TenantSubscriptionResponse{
		ID:        s.ID,
		TenantID:  s.TenantID,
		ItemID:    s.ItemID,
		Status:    s.Status,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}

// IdeSyncManifest returns full marketplace item payloads for all active tenant subscriptions (IDE pull).
func (s *Service) IdeSyncManifest(ctx context.Context, tenantID string) ([]*dto.MarketplaceItemResponse, error) {
	items, err := s.repo.ListActiveSubscribedItemsForTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]*dto.MarketplaceItemResponse, 0, len(items))
	for _, it := range items {
		out = append(out, itemToDTO(it))
	}
	return out, nil
}

// GetItemDownloadURL returns a download URL for a published, subscribed marketplace item.
// If the item payload JSON contains "download_url", that value is used; otherwise a placeholder URL is returned.
func (s *Service) GetItemDownloadURL(ctx context.Context, tenantID, itemID string) (*dto.MarketplaceItemDownloadResponse, error) {
	item, err := s.repo.GetMarketplaceItemByID(ctx, itemID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrMarketplaceItemNotFound
		}
		return nil, err
	}
	if item.Status != domain.MarketplaceItemStatusPublished {
		return nil, domain.ErrMarketplaceItemNotPublished
	}
	sub, err := s.repo.GetTenantSubscriptionByTenantAndItem(ctx, tenantID, itemID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrMarketplaceSubscriptionNotFound
		}
		return nil, err
	}
	if sub.Status != domain.TenantSubscriptionStatusActive {
		return nil, domain.ErrMarketplaceSubscriptionNotFound
	}
	var payload struct {
		DownloadURL string `json:"download_url"`
	}
	if item.Payload != "" {
		_ = json.Unmarshal([]byte(item.Payload), &payload)
	}
	url := strings.TrimSpace(payload.DownloadURL)
	if url == "" {
		url = fmt.Sprintf("https://example.com/marketplace/plugins/%s.zip", itemID)
	}
	return &dto.MarketplaceItemDownloadResponse{DownloadURL: url}, nil
}
