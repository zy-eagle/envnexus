package watchlist

import (
	"context"
	"log/slog"
)

// RemediationSuggestion holds a generated remediation plan reference.
type RemediationSuggestion struct {
	PlanID    string `json:"plan_id"`
	Summary   string `json:"summary"`
	RiskLevel string `json:"risk_level"`
	StepCount int    `json:"step_count"`
}

// AlertRemediator generates remediation suggestions from alerts.
// This is implemented by a thin adapter in bootstrap that wraps the remediation.Planner.
type AlertRemediator interface {
	SuggestRemediation(ctx context.Context, alertSummary string, severity string) (*RemediationSuggestion, error)
}

type Alerter struct {
	store      *Store
	scheduler  *Scheduler
	remediator AlertRemediator
}

func NewAlerter(store *Store, scheduler *Scheduler) *Alerter {
	return &Alerter{
		store:     store,
		scheduler: scheduler,
	}
}

func (a *Alerter) SetRemediator(r AlertRemediator) {
	a.remediator = r
}

// HandleAlert processes a new alert: optionally generates a remediation suggestion
// and returns it for the caller to present to the user.
func (a *Alerter) HandleAlert(ctx context.Context, alert *WatchAlert) *RemediationSuggestion {
	if a.remediator == nil {
		return nil
	}

	if alert.Severity != SeverityCritical && alert.Severity != SeverityWarning {
		return nil
	}

	slog.Info("[Alerter] generating remediation suggestion",
		"alert", alert.ID, "item", alert.ItemName, "severity", alert.Severity)

	summary := "Watchlist alert: " + alert.ItemName + " — " + alert.Message

	suggestion, err := a.remediator.SuggestRemediation(ctx, summary, string(alert.Severity))
	if err != nil {
		slog.Warn("[Alerter] failed to generate remediation suggestion",
			"alert", alert.ID, "error", err)
		return nil
	}

	slog.Info("[Alerter] remediation suggestion generated",
		"alert", alert.ID, "plan", suggestion.PlanID, "steps", suggestion.StepCount)
	return suggestion
}

// VerifyAfterRemediation re-runs the watch item check to verify the fix worked.
func (a *Alerter) VerifyAfterRemediation(ctx context.Context, watchItemID string) error {
	return a.scheduler.RunNow(ctx, watchItemID)
}
