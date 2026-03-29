package domain

import "time"

type GovernanceBaseline struct {
	ID           string
	DeviceID     string
	TenantID     string
	SnapshotJSON string
	CapturedAt   time.Time
}

func (g *GovernanceBaseline) TableName() string { return "governance_baselines" }

type GovernanceDrift struct {
	ID            string
	DeviceID      string
	TenantID      string
	BaselineID    *string
	DriftType     string
	KeyName       string
	ExpectedValue *string
	ActualValue   *string
	Severity      string
	DetectedAt    time.Time
	ResolvedAt    *time.Time
}

func (g *GovernanceDrift) TableName() string { return "governance_drifts" }
