package domain

import (
	"encoding/json"
	"sort"
	"time"
)

// Predefined platform roles
const (
	RolePlatformSuperAdmin = "platform_super_admin"
	RoleTenantAdmin        = "tenant_admin"
	RoleSecurityAuditor    = "security_auditor"
	RoleOpsOperator        = "ops_operator"
	RoleReadOnlyObserver   = "read_only_observer"
)

// Permission constants
const (
	PermManageTenants        = "tenants:manage"
	PermViewTenants          = "tenants:view"
	PermManageUsers          = "users:manage"
	PermManageRoles          = "roles:manage"
	PermManageProfiles       = "profiles:manage"
	PermViewProfiles         = "profiles:view"
	PermManageDevices        = "devices:manage"
	PermViewDevices          = "devices:view"
	PermRevokeDevices        = "devices:revoke"
	PermManageSessions       = "sessions:manage"
	PermViewSessions         = "sessions:view"
	PermApproveActions       = "approvals:approve"
	PermViewAudit            = "audit:view"
	PermManagePackages       = "packages:manage"
	PermManageWebhooks       = "webhooks:manage"
	PermViewMetrics          = "metrics:view"
	PermManageLicenses       = "licenses:manage"
	PermCommandEmergency     = "command:emergency"
	PermCommandBypassApproval = "command:bypass_approval"
)

// assignablePermissions is the canonical list for role UIs (tenant-scoped roles).
var assignablePermissions = []string{
	PermViewTenants, PermManageTenants,
	PermManageUsers, PermManageRoles,
	PermManageProfiles, PermViewProfiles,
	PermManageDevices, PermViewDevices, PermRevokeDevices,
	PermManageSessions, PermViewSessions,
	PermApproveActions,
	PermViewAudit,
	PermManagePackages, PermManageWebhooks,
	PermViewMetrics, PermManageLicenses,
	PermCommandEmergency,
	PermCommandBypassApproval,
}

// AssignablePermissions returns every permission that may be granted on a tenant role,
// sorted lexicographically for stable UI ordering.
func AssignablePermissions() []string {
	out := make([]string, len(assignablePermissions))
	copy(out, assignablePermissions)
	sort.Strings(out)
	return out
}

// DefaultRolePermissions maps role name → permissions slice
var DefaultRolePermissions = map[string][]string{
	RolePlatformSuperAdmin: {
		PermManageTenants, PermViewTenants,
		PermManageUsers, PermManageRoles,
		PermManageProfiles, PermViewProfiles,
		PermManageDevices, PermViewDevices, PermRevokeDevices,
		PermManageSessions, PermViewSessions,
		PermApproveActions, PermViewAudit,
		PermManagePackages, PermManageWebhooks,
		PermViewMetrics, PermManageLicenses,
	},
	RoleTenantAdmin: {
		PermViewTenants,
		PermManageUsers,
		PermManageProfiles, PermViewProfiles,
		PermManageDevices, PermViewDevices, PermRevokeDevices,
		PermManageSessions, PermViewSessions,
		PermApproveActions, PermViewAudit,
		PermManagePackages, PermManageWebhooks,
		PermViewMetrics,
	},
	RoleSecurityAuditor: {
		PermViewTenants, PermViewProfiles,
		PermViewDevices, PermViewSessions,
		PermApproveActions, PermViewAudit,
	},
	RoleOpsOperator: {
		PermViewTenants, PermViewProfiles,
		PermManageDevices, PermViewDevices,
		PermManageSessions, PermViewSessions,
		PermApproveActions, PermViewAudit,
		PermManagePackages,
	},
	RoleReadOnlyObserver: {
		PermViewTenants, PermViewProfiles,
		PermViewDevices, PermViewSessions,
		PermViewAudit,
	},
}

type Role struct {
	ID              string    `json:"id"              gorm:"primaryKey;size:26"`
	TenantID        string    `json:"tenant_id"       gorm:"size:26;not null;index"`
	Name            string    `json:"name"            gorm:"size:64;not null"`
	PermissionsJSON string    `json:"-"               gorm:"type:json;not null"`
	Status          string    `json:"status"          gorm:"size:32;not null;default:active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (r *Role) Permissions() []string {
	var perms []string
	if err := json.Unmarshal([]byte(r.PermissionsJSON), &perms); err != nil {
		return nil
	}
	return perms
}

func (r *Role) HasPermission(perm string) bool {
	for _, p := range r.Permissions() {
		if p == perm {
			return true
		}
	}
	return false
}

type RoleBinding struct {
	ID        string    `json:"id"         gorm:"primaryKey;size:26"`
	TenantID  string    `json:"tenant_id"  gorm:"size:26;not null;index"`
	UserID    string    `json:"user_id"    gorm:"size:26;not null;index"`
	RoleID    string    `json:"role_id"    gorm:"size:26;not null"`
	GrantedBy string    `json:"granted_by" gorm:"size:26"`
	CreatedAt time.Time `json:"created_at"`

	Role *Role `json:"role,omitempty" gorm:"foreignKey:RoleID"`
}
