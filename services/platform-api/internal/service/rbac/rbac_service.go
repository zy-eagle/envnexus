package rbac

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/domain"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/repository"
)

type Service struct {
	roleRepo    repository.RoleRepository
	bindingRepo repository.RoleBindingRepository
}

func NewService(roleRepo repository.RoleRepository, bindingRepo repository.RoleBindingRepository) *Service {
	return &Service{roleRepo: roleRepo, bindingRepo: bindingRepo}
}

// SeedDefaultRoles creates the 5 preset roles for a tenant if they don't exist.
func (s *Service) SeedDefaultRoles(ctx context.Context, tenantID string) error {
	for name, perms := range domain.DefaultRolePermissions {
		existing, err := s.roleRepo.GetByName(ctx, tenantID, name)
		if err != nil {
			return fmt.Errorf("check role %s: %w", name, err)
		}
		if existing != nil {
			continue
		}
		permsJSON, _ := json.Marshal(perms)
		role := &domain.Role{
			ID:              ulid.Make().String(),
			TenantID:        tenantID,
			Name:            name,
			PermissionsJSON: string(permsJSON),
			Status:          "active",
		}
		if err := s.roleRepo.Create(ctx, role); err != nil {
			return fmt.Errorf("create role %s: %w", name, err)
		}
		slog.Info("seeded default role", "tenant_id", tenantID, "role", name)
	}
	return nil
}

// GetUserPermissions returns all permissions for a user across their role bindings.
func (s *Service) GetUserPermissions(ctx context.Context, userID string) ([]string, error) {
	bindings, err := s.bindingRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	permSet := make(map[string]struct{})
	for _, b := range bindings {
		if b.Role == nil {
			continue
		}
		for _, p := range b.Role.Permissions() {
			permSet[p] = struct{}{}
		}
	}
	perms := make([]string, 0, len(permSet))
	for p := range permSet {
		perms = append(perms, p)
	}
	return perms, nil
}

// HasPermission checks if a user has a specific permission.
func (s *Service) HasPermission(ctx context.Context, userID, permission string) (bool, error) {
	perms, err := s.GetUserPermissions(ctx, userID)
	if err != nil {
		return false, err
	}
	for _, p := range perms {
		if p == permission {
			return true, nil
		}
	}
	return false, nil
}

// ListRoles lists all roles for a tenant.
func (s *Service) ListRoles(ctx context.Context, tenantID string) ([]*domain.Role, error) {
	return s.roleRepo.ListByTenant(ctx, tenantID)
}

// GetRole fetches a single role.
func (s *Service) GetRole(ctx context.Context, id string) (*domain.Role, error) {
	return s.roleRepo.GetByID(ctx, id)
}

// CreateRole creates a custom role.
func (s *Service) CreateRole(ctx context.Context, tenantID, name string, perms []string) (*domain.Role, error) {
	permsJSON, _ := json.Marshal(perms)
	role := &domain.Role{
		ID:              ulid.Make().String(),
		TenantID:        tenantID,
		Name:            name,
		PermissionsJSON: string(permsJSON),
		Status:          "active",
	}
	if err := s.roleRepo.Create(ctx, role); err != nil {
		return nil, err
	}
	return role, nil
}

// UpdateRole updates a role's permissions.
func (s *Service) UpdateRole(ctx context.Context, id string, perms []string) (*domain.Role, error) {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, domain.ErrNotFound
	}
	permsJSON, _ := json.Marshal(perms)
	role.PermissionsJSON = string(permsJSON)
	role.UpdatedAt = time.Now()
	if err := s.roleRepo.Update(ctx, role); err != nil {
		return nil, err
	}
	return role, nil
}

// DeleteRole removes a role.
func (s *Service) DeleteRole(ctx context.Context, id string) error {
	return s.roleRepo.Delete(ctx, id)
}

// BindRole assigns a role to a user.
func (s *Service) BindRole(ctx context.Context, tenantID, userID, roleID, grantedBy string) (*domain.RoleBinding, error) {
	rb := &domain.RoleBinding{
		ID:        ulid.Make().String(),
		TenantID:  tenantID,
		UserID:    userID,
		RoleID:    roleID,
		GrantedBy: grantedBy,
	}
	if err := s.bindingRepo.Create(ctx, rb); err != nil {
		return nil, err
	}
	return rb, nil
}

// UnbindRole removes a role from a user.
func (s *Service) UnbindRole(ctx context.Context, userID, roleID string) error {
	return s.bindingRepo.DeleteByUserAndRole(ctx, userID, roleID)
}

// ListBindings lists all bindings for a tenant.
func (s *Service) ListBindings(ctx context.Context, tenantID string) ([]*domain.RoleBinding, error) {
	return s.bindingRepo.ListByTenant(ctx, tenantID)
}
