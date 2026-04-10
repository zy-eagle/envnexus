package watchlist

import (
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func setupTestStore(t *testing.T) *Store {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })

	s := NewStore(db)
	if err := s.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return s
}

func TestStore_SaveAndGetWatchItem(t *testing.T) {
	s := setupTestStore(t)

	item := &WatchItem{
		ID:          "test-1",
		Name:        "Test Item",
		Description: "A test watch item",
		Source:      SourceUser,
		ToolName:    "ping_host",
		ToolParams:  map[string]interface{}{"host": "8.8.8.8"},
		Condition: WatchCondition{
			Type: CondReachable,
		},
		Interval:  5 * time.Minute,
		Enabled:   true,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.SaveWatchItem(item); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := s.GetWatchItem("test-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil item")
	}
	if got.Name != "Test Item" {
		t.Errorf("expected name 'Test Item', got %q", got.Name)
	}
	if got.ToolName != "ping_host" {
		t.Errorf("expected tool 'ping_host', got %q", got.ToolName)
	}
	if !got.Enabled {
		t.Error("expected enabled=true")
	}
}

func TestStore_ListWatchItems(t *testing.T) {
	s := setupTestStore(t)

	for _, id := range []string{"a", "b", "c"} {
		s.SaveWatchItem(&WatchItem{
			ID: id, Name: id, Source: SourceBuiltin, ToolName: "test",
			Condition: WatchCondition{Type: CondReachable},
			Interval: time.Minute, Enabled: true, CreatedAt: time.Now().UTC(),
		})
	}

	items, err := s.ListWatchItems("")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 3 {
		t.Errorf("expected 3 items, got %d", len(items))
	}

	items, err = s.ListWatchItems("builtin")
	if err != nil {
		t.Fatalf("list by source: %v", err)
	}
	if len(items) != 3 {
		t.Errorf("expected 3 builtin items, got %d", len(items))
	}
}

func TestStore_DeleteWatchItem(t *testing.T) {
	s := setupTestStore(t)

	s.SaveWatchItem(&WatchItem{
		ID: "del-1", Name: "Delete Me", Source: SourceUser, ToolName: "test",
		Condition: WatchCondition{Type: CondReachable},
		Interval: time.Minute, Enabled: true, CreatedAt: time.Now().UTC(),
	})

	if err := s.DeleteWatchItem("del-1"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	got, err := s.GetWatchItem("del-1")
	if err != nil {
		t.Fatalf("get after delete: %v", err)
	}
	if got != nil {
		t.Error("expected nil after delete")
	}
}

func TestStore_EnableDisable(t *testing.T) {
	s := setupTestStore(t)

	s.SaveWatchItem(&WatchItem{
		ID: "toggle-1", Name: "Toggle", Source: SourceUser, ToolName: "test",
		Condition: WatchCondition{Type: CondReachable},
		Interval: time.Minute, Enabled: true, CreatedAt: time.Now().UTC(),
	})

	s.SetWatchItemEnabled("toggle-1", false)
	got, _ := s.GetWatchItem("toggle-1")
	if got.Enabled {
		t.Error("expected disabled")
	}

	s.SetWatchItemEnabled("toggle-1", true)
	got, _ = s.GetWatchItem("toggle-1")
	if !got.Enabled {
		t.Error("expected enabled")
	}
}

func TestStore_Alerts(t *testing.T) {
	s := setupTestStore(t)

	alert := &WatchAlert{
		ID:          "alert-1",
		WatchItemID: "item-1",
		ItemName:    "Test Alert",
		Severity:    SeverityWarning,
		Message:     "Something is wrong",
	}

	if err := s.SaveAlert(alert); err != nil {
		t.Fatalf("save alert: %v", err)
	}

	resolved := false
	alerts, err := s.ListAlerts(&resolved, 10)
	if err != nil {
		t.Fatalf("list alerts: %v", err)
	}
	if len(alerts) != 1 {
		t.Errorf("expected 1 alert, got %d", len(alerts))
	}

	if err := s.ResolveAlert("alert-1"); err != nil {
		t.Fatalf("resolve: %v", err)
	}

	alerts, err = s.ListAlerts(&resolved, 10)
	if err != nil {
		t.Fatalf("list after resolve: %v", err)
	}
	if len(alerts) != 0 {
		t.Errorf("expected 0 unresolved alerts, got %d", len(alerts))
	}
}

func TestStore_CountAlertsBySeverity(t *testing.T) {
	s := setupTestStore(t)

	s.SaveAlert(&WatchAlert{ID: "a1", WatchItemID: "i1", ItemName: "A", Severity: SeverityWarning, Message: "w"})
	s.SaveAlert(&WatchAlert{ID: "a2", WatchItemID: "i1", ItemName: "A", Severity: SeverityCritical, Message: "c"})
	s.SaveAlert(&WatchAlert{ID: "a3", WatchItemID: "i1", ItemName: "A", Severity: SeverityInfo, Message: "i"})

	info, warning, critical, err := s.CountAlertsBySeverity()
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if info != 1 || warning != 1 || critical != 1 {
		t.Errorf("expected 1/1/1, got %d/%d/%d", info, warning, critical)
	}
}
