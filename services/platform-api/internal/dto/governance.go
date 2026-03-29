package dto

import "time"

type GovernanceBaselineResponse struct {
	ID           string    `json:"id"`
	DeviceID     string    `json:"device_id"`
	TenantID     string    `json:"tenant_id"`
	SnapshotJSON string    `json:"snapshot_json"`
	CapturedAt   time.Time `json:"captured_at"`
}

type GovernanceDriftResponse struct {
	ID            string     `json:"id"`
	DeviceID      string     `json:"device_id"`
	TenantID      string     `json:"tenant_id"`
	BaselineID    *string    `json:"baseline_id"`
	DriftType     string     `json:"drift_type"`
	KeyName       string     `json:"key_name"`
	ExpectedValue *string    `json:"expected_value"`
	ActualValue   *string    `json:"actual_value"`
	Severity      string     `json:"severity"`
	DetectedAt    time.Time  `json:"detected_at"`
	ResolvedAt    *time.Time `json:"resolved_at"`
}

type ReportBaselineRequest struct {
	SnapshotJSON string `json:"snapshot_json" binding:"required"`
}

type ReportDriftsRequest struct {
	Drifts []ReportDriftItem `json:"drifts" binding:"required,min=1"`
}

type ReportDriftItem struct {
	DriftType     string  `json:"drift_type" binding:"required"`
	KeyName       string  `json:"key_name" binding:"required"`
	ExpectedValue *string `json:"expected_value"`
	ActualValue   *string `json:"actual_value"`
	Severity      string  `json:"severity" binding:"required,oneof=low medium high critical"`
}
