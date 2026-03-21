package repository

import (
	"context"

	"github.com/zy-eagle/envnexus/services/platform-api/internal/domain"
)

type TenantRepository interface {
	Create(ctx context.Context, tenant *domain.Tenant) error
	GetByID(ctx context.Context, id string) (*domain.Tenant, error)
	GetBySlug(ctx context.Context, slug string) (*domain.Tenant, error)
}

// MemoryTenantRepository is a simple in-memory implementation for MVP
type MemoryTenantRepository struct {
	tenants map[string]*domain.Tenant
}

func NewMemoryTenantRepository() *MemoryTenantRepository {
	return &MemoryTenantRepository{
		tenants: make(map[string]*domain.Tenant),
	}
}

func (r *MemoryTenantRepository) Create(ctx context.Context, tenant *domain.Tenant) error {
	r.tenants[tenant.ID] = tenant
	return nil
}

func (r *MemoryTenantRepository) GetByID(ctx context.Context, id string) (*domain.Tenant, error) {
	if t, ok := r.tenants[id]; ok {
		return t, nil
	}
	return nil, nil // Should return a proper error like ErrNotFound
}

func (r *MemoryTenantRepository) GetBySlug(ctx context.Context, slug string) (*domain.Tenant, error) {
	for _, t := range r.tenants {
		if t.Slug == slug {
			return t, nil
		}
	}
	return nil, nil
}
