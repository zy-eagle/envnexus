package governance

import (
	"context"
	"testing"
)

func TestEngine_NoWatchlistManager_BehaviorUnchanged(t *testing.T) {
	e := NewEngine()

	status := e.GetStatus()
	if status.HasBaseline {
		t.Error("expected no baseline without store")
	}
	if status.DriftCount != 0 {
		t.Error("expected zero drift count without store")
	}
}

func TestEngine_GetHealthScore_NilWatchlist(t *testing.T) {
	e := NewEngine()

	score, err := e.GetHealthScore()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if score.Score != 100 {
		t.Errorf("expected score 100 without watchlist, got %d", score.Score)
	}
}

func TestEngine_GetAlerts_NilWatchlist(t *testing.T) {
	e := NewEngine()

	alerts, err := e.GetAlerts(nil, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if alerts != nil {
		t.Error("expected nil alerts without watchlist")
	}
}

func TestEngine_GetWatchlistManager_Nil(t *testing.T) {
	e := NewEngine()

	wm := e.GetWatchlistManager()
	if wm != nil {
		t.Error("expected nil watchlist manager")
	}
}

func TestEngine_Start_NilWatchlist_NoError(t *testing.T) {
	e := NewEngine()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	e.Start(ctx)
}

func TestEngine_CaptureBaseline_NoStore(t *testing.T) {
	e := NewEngine()

	baseline, err := e.CaptureBaseline()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if baseline == nil {
		t.Fatal("expected non-nil baseline")
	}
	if baseline.Hostname == "" {
		t.Error("expected non-empty hostname")
	}
}

func TestEngine_DetectDrift_NoStore(t *testing.T) {
	e := NewEngine()

	drifts, err := e.DetectDrift()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if drifts != nil {
		t.Error("expected nil drifts without store")
	}
}

func TestEngine_RunBaselineCheck_NoStore(t *testing.T) {
	e := NewEngine()

	e.RunBaselineCheck(context.Background())
}
