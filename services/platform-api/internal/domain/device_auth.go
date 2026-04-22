package domain

import (
	"context"
	"time"
)

// DeviceAuthCodeStatus tracks the device authorization (RFC 8628) flow.
type DeviceAuthCodeStatus string

const (
	DeviceAuthCodeStatusPending    DeviceAuthCodeStatus = "pending"
	DeviceAuthCodeStatusAuthorized DeviceAuthCodeStatus = "authorized"
	DeviceAuthCodeStatusDenied     DeviceAuthCodeStatus = "denied"
	DeviceAuthCodeStatusExpired    DeviceAuthCodeStatus = "expired"
)

// DeviceAuthCode stores a short-lived device authorization session (may be backed by Redis in production).
type DeviceAuthCode struct {
	DeviceCode string               `json:"device_code"  gorm:"primaryKey;size:512;not null"`
	UserCode   string               `json:"user_code"    gorm:"size:32;not null;index"`
	Status     DeviceAuthCodeStatus `json:"status"       gorm:"size:32;not null;index;default:pending"`
	ExpiresAt  time.Time            `json:"expires_at"   gorm:"not null;index"`
	UserID     *string              `json:"user_id"      gorm:"size:26"`
	TenantID   *string              `json:"tenant_id"    gorm:"size:26"`
	DeviceInfo string               `json:"device_info"  gorm:"type:json"`
	CreatedAt  time.Time            `json:"created_at"`
	UpdatedAt  time.Time            `json:"updated_at"`
}

// TableName returns the GORM table name for DeviceAuthCode.
func (DeviceAuthCode) TableName() string { return "device_auth_codes" }

// IdeClientToken stores hashed access/refresh material for IDE clients (tokens are never stored in plaintext).
type IdeClientToken struct {
	ID               string     `json:"id"                 gorm:"primaryKey;size:26"`
	UserID           string     `json:"user_id"            gorm:"size:26;not null;index"`
	TenantID         string     `json:"tenant_id"          gorm:"size:26;not null;index"`
	Name             string     `json:"name"               gorm:"size:255;not null"`
	AccessTokenHash  string     `json:"access_token_hash"  gorm:"size:64;not null;index"`
	RefreshTokenHash string     `json:"refresh_token_hash" gorm:"size:64;not null;index"`
	AccessExpiresAt  time.Time  `json:"access_expires_at"  gorm:"not null;index"`
	RefreshExpiresAt time.Time  `json:"refresh_expires_at" gorm:"not null;index"`
	LastUsedAt       *time.Time `json:"last_used_at"       gorm:"index"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// TableName returns the GORM table name for IdeClientToken.
func (IdeClientToken) TableName() string { return "ide_client_tokens" }

// DeviceAuthRepository persists device authorization sessions and IDE client tokens.
type DeviceAuthRepository interface {
	CreateDeviceAuthCode(ctx context.Context, code *DeviceAuthCode) error
	GetDeviceAuthCodeByDeviceCode(ctx context.Context, deviceCode string) (*DeviceAuthCode, error)
	GetDeviceAuthCodeByUserCode(ctx context.Context, userCode string) (*DeviceAuthCode, error)
	ListDeviceAuthCodes(ctx context.Context, page, pageSize int) ([]*DeviceAuthCode, int64, error)
	UpdateDeviceAuthCode(ctx context.Context, code *DeviceAuthCode) error
	DeleteDeviceAuthCode(ctx context.Context, deviceCode string) error

	CreateIdeClientToken(ctx context.Context, t *IdeClientToken) error
	GetIdeClientTokenByID(ctx context.Context, id string) (*IdeClientToken, error)
	GetIdeClientTokenByAccessTokenHash(ctx context.Context, accessTokenHash string) (*IdeClientToken, error)
	GetIdeClientTokenByRefreshTokenHash(ctx context.Context, refreshTokenHash string) (*IdeClientToken, error)
	ListIdeClientTokensByUserID(ctx context.Context, userID string, page, pageSize int) ([]*IdeClientToken, int64, error)
	ListIdeClientTokensByTenantID(ctx context.Context, tenantID string, page, pageSize int) ([]*IdeClientToken, int64, error)
	UpdateIdeClientToken(ctx context.Context, t *IdeClientToken) error
	DeleteIdeClientToken(ctx context.Context, id string) error
}
