package watchlist

import (
	"encoding/json"
	"testing"
)

func TestRuleToWatchItem_Threshold(t *testing.T) {
	cond, _ := json.Marshal(map[string]interface{}{
		"tool_name":   "disk_usage",
		"tool_params": map[string]interface{}{"path": "/"},
		"type":        "threshold",
		"json_path":   "$.usage_percent",
		"operator":    "gt",
		"threshold":   90,
		"interval":    "30s",
	})
	rule := PlatformRule{
		ID:            "rule-1",
		Name:          "Disk alert",
		Enabled:       true,
		ConditionJSON: string(cond),
	}
	item, ok := ruleToWatchItem(rule)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if item.ToolName != "disk_usage" {
		t.Errorf("tool_name: got %q", item.ToolName)
	}
	if item.Condition.Type != CondThreshold {
		t.Errorf("condition type: got %q", item.Condition.Type)
	}
	if item.Condition.Operator != OpGT {
		t.Errorf("operator: got %q", item.Condition.Operator)
	}
	if item.Source != SourcePlatform {
		t.Errorf("source: got %q", item.Source)
	}
	if item.Interval.Seconds() != 30 {
		t.Errorf("interval: got %v", item.Interval)
	}
}

func TestRuleToWatchItem_EmptyCondition(t *testing.T) {
	_, ok := ruleToWatchItem(PlatformRule{ID: "x"})
	if ok {
		t.Error("expected false for empty condition")
	}
}

func TestRuleToWatchItem_MissingToolName(t *testing.T) {
	cond, _ := json.Marshal(map[string]interface{}{"type": "exists"})
	_, ok := ruleToWatchItem(PlatformRule{ID: "y", ConditionJSON: string(cond)})
	if ok {
		t.Error("expected false when tool_name missing")
	}
}

func TestTruncateBody(t *testing.T) {
	if truncateBody("abc", 10) != "abc" {
		t.Error("short should be returned as-is")
	}
	if truncateBody("0123456789", 5) != "01234..." {
		t.Error("long should be truncated with ellipsis")
	}
}
