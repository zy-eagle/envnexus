package license

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
	"gorm.io/gorm"
)

// LicenseRow mirrors the licenses table.
type LicenseRow struct {
	ID           string     `gorm:"primaryKey;size:26"`
	TenantID     string     `gorm:"size:26;not null;index"`
	LicenseKey   string     `gorm:"size:512;not null;uniqueIndex"`
	PlanCode     string     `gorm:"size:32;not null;default:enterprise"`
	MaxDevices   int        `gorm:"not null;default:100"`
	FeaturesJSON *string    `gorm:"type:json"`
	IssuedAt     time.Time
	ExpiresAt    time.Time
	Status       string     `gorm:"size:32;not null;default:active"`
	CreatedAt    time.Time
}

func (LicenseRow) TableName() string { return "licenses" }

type ValidateLicenseResult struct {
	Valid      bool
	TenantID   string
	PlanCode   string
	MaxDevices int
	ExpiresAt  time.Time
	Features   []string
}

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// Activate activates a license key for a tenant.
func (s *Service) Activate(ctx context.Context, tenantID, licenseKey string) (*LicenseRow, error) {
	// Parse key format: ENX-{plan}-{maxDevices}-{expiry(YYYYMM)}-{checksum}
	if err := s.verifyKeyFormat(licenseKey); err != nil {
		return nil, fmt.Errorf("invalid license key format: %w", err)
	}

	plan, maxDevices, expiresAt, err := s.parseKey(licenseKey)
	if err != nil {
		return nil, fmt.Errorf("parse license key: %w", err)
	}

	row := &LicenseRow{
		ID:         ulid.Make().String(),
		TenantID:   tenantID,
		LicenseKey: licenseKey,
		PlanCode:   plan,
		MaxDevices: maxDevices,
		IssuedAt:   time.Now(),
		ExpiresAt:  expiresAt,
		Status:     "active",
	}
	if err := s.db.WithContext(ctx).Create(row).Error; err != nil {
		return nil, err
	}
	slog.Info("License activated", "tenant_id", tenantID, "plan", plan, "max_devices", maxDevices, "expires_at", expiresAt)
	return row, nil
}

// Validate checks if a tenant has a valid active license.
func (s *Service) Validate(ctx context.Context, tenantID string) (*ValidateLicenseResult, error) {
	var row LicenseRow
	err := s.db.WithContext(ctx).
		Where("tenant_id = ? AND status = ? AND expires_at > ?", tenantID, "active", time.Now()).
		Order("expires_at DESC").
		First(&row).Error

	if err == gorm.ErrRecordNotFound {
		// No license — return a default trial license
		return &ValidateLicenseResult{
			Valid:      true,
			TenantID:   tenantID,
			PlanCode:   "trial",
			MaxDevices: 5,
			ExpiresAt:  time.Now().AddDate(0, 1, 0),
		}, nil
	}
	if err != nil {
		return nil, err
	}

	return &ValidateLicenseResult{
		Valid:      true,
		TenantID:   tenantID,
		PlanCode:   row.PlanCode,
		MaxDevices: row.MaxDevices,
		ExpiresAt:  row.ExpiresAt,
	}, nil
}

// GetForTenant returns the license for a tenant.
func (s *Service) GetForTenant(ctx context.Context, tenantID string) (*LicenseRow, error) {
	var row LicenseRow
	err := s.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("expires_at DESC").
		First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &row, err
}

// Revoke sets a license to revoked status.
func (s *Service) Revoke(ctx context.Context, licenseID string) error {
	return s.db.WithContext(ctx).
		Table("licenses").
		Where("id = ?", licenseID).
		Updates(map[string]interface{}{"status": "revoked", "updated_at": time.Now()}).Error
}

// verifyKeyFormat validates basic key structure.
func (s *Service) verifyKeyFormat(key string) error {
	parts := strings.Split(key, "-")
	if len(parts) < 5 || parts[0] != "ENX" {
		return fmt.Errorf("key must be in format ENX-{plan}-{max_devices}-{YYYYMM}-{checksum}")
	}
	return nil
}

// parseKey extracts plan, max devices, and expiry from key.
// Format: ENX-{PLAN}-{MaxDevices}-{YYYYMM}-{checksum}
func (s *Service) parseKey(key string) (plan string, maxDevices int, expiresAt time.Time, err error) {
	parts := strings.Split(key, "-")
	if len(parts) < 5 {
		return "", 0, time.Time{}, fmt.Errorf("invalid key format")
	}

	plan = strings.ToLower(parts[1])

	_, err = fmt.Sscanf(parts[2], "%d", &maxDevices)
	if err != nil {
		return "", 0, time.Time{}, fmt.Errorf("invalid max_devices in key")
	}

	expiresAt, err = time.Parse("200601", parts[3])
	if err != nil {
		return "", 0, time.Time{}, fmt.Errorf("invalid expiry date in key (expected YYYYMM)")
	}
	// Expire at end of month
	expiresAt = expiresAt.AddDate(0, 1, 0).Add(-time.Second)

	// Verify checksum: SHA256 of ENX-{plan}-{max_devices}-{YYYYMM}
	base := strings.Join(parts[:4], "-")
	expectedChecksum := computeChecksum(base)
	if !strings.HasPrefix(expectedChecksum, parts[4]) {
		return "", 0, time.Time{}, fmt.Errorf("license key checksum mismatch")
	}

	return plan, maxDevices, expiresAt, nil
}

func computeChecksum(s string) string {
	h := sha256.Sum256([]byte(s))
	return strings.ToUpper(hex.EncodeToString(h[:4]))
}

// GenerateKey generates a valid license key for a given plan, max_devices, and expiry (YYYYMM).
// This would normally be done by a key signing service; exposed here for operator tooling.
func GenerateKey(plan string, maxDevices int, expiryYYYYMM string) string {
	plan = strings.ToUpper(plan)
	base := fmt.Sprintf("ENX-%s-%d-%s", plan, maxDevices, expiryYYYYMM)
	checksum := computeChecksum(base)
	return base + "-" + checksum
}
