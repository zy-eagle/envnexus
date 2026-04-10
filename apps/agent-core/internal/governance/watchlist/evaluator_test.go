package watchlist

import (
	"testing"
)

func TestEvalThreshold_GT(t *testing.T) {
	cond := WatchCondition{
		Type:      CondThreshold,
		JSONPath:  "usage_percent",
		Operator:  OpGT,
		Threshold: float64(90),
	}
	output := map[string]interface{}{"usage_percent": float64(95)}

	result, err := Evaluate(cond, output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Triggered {
		t.Error("expected triggered for 95 > 90")
	}
}

func TestEvalThreshold_NotTriggered(t *testing.T) {
	cond := WatchCondition{
		Type:      CondThreshold,
		JSONPath:  "usage_percent",
		Operator:  OpGT,
		Threshold: float64(90),
	}
	output := map[string]interface{}{"usage_percent": float64(50)}

	result, err := Evaluate(cond, output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Triggered {
		t.Error("expected not triggered for 50 > 90")
	}
}

func TestEvalThreshold_LT(t *testing.T) {
	cond := WatchCondition{
		Type:      CondThreshold,
		JSONPath:  "days_until_expiry",
		Operator:  OpLT,
		Threshold: float64(30),
	}
	output := map[string]interface{}{"days_until_expiry": float64(10)}

	result, err := Evaluate(cond, output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Triggered {
		t.Error("expected triggered for 10 < 30")
	}
}

func TestEvalThreshold_NestedPath(t *testing.T) {
	cond := WatchCondition{
		Type:      CondThreshold,
		JSONPath:  "memory.used_percent",
		Operator:  OpGT,
		Threshold: float64(90),
	}
	output := map[string]interface{}{
		"memory": map[string]interface{}{
			"used_percent": float64(92),
		},
	}

	result, err := Evaluate(cond, output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Triggered {
		t.Error("expected triggered for nested 92 > 90")
	}
}

func TestEvalThreshold_MissingKey(t *testing.T) {
	cond := WatchCondition{
		Type:      CondThreshold,
		JSONPath:  "nonexistent",
		Operator:  OpGT,
		Threshold: float64(90),
	}
	output := map[string]interface{}{"usage_percent": float64(95)}

	result, err := Evaluate(cond, output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Triggered {
		t.Error("expected triggered when key is missing (failure mode)")
	}
}

func TestEvalReachable_Success(t *testing.T) {
	output := map[string]interface{}{"status": "success"}

	result, err := Evaluate(WatchCondition{Type: CondReachable}, output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Triggered {
		t.Error("expected not triggered for status=success")
	}
}

func TestEvalReachable_Failure(t *testing.T) {
	output := map[string]interface{}{"status": "failed", "error": "connection refused"}

	result, err := Evaluate(WatchCondition{Type: CondReachable}, output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Triggered {
		t.Error("expected triggered for status=failed")
	}
	if result.Message == "" {
		t.Error("expected non-empty message")
	}
}

func TestEvalContains_Match(t *testing.T) {
	cond := WatchCondition{
		Type:     CondContains,
		JSONPath: "output",
		Operator: OpContains,
		Pattern:  "ERROR",
	}
	output := map[string]interface{}{"output": "some ERROR occurred"}

	result, err := Evaluate(cond, output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Triggered {
		t.Error("expected triggered for string containing ERROR")
	}
}

func TestEvalContains_NotContains(t *testing.T) {
	cond := WatchCondition{
		Type:     CondContains,
		JSONPath: "output",
		Operator: OpNotContains,
		Pattern:  "ERROR",
	}
	output := map[string]interface{}{"output": "all good"}

	result, err := Evaluate(cond, output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Triggered {
		t.Error("expected triggered for string not containing ERROR with not_contains operator")
	}
}

func TestEvalExists_Present(t *testing.T) {
	cond := WatchCondition{
		Type:     CondExists,
		JSONPath: "pid",
	}
	output := map[string]interface{}{"pid": 1234}

	result, err := Evaluate(cond, output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Triggered {
		t.Error("expected not triggered when value exists (default: should exist)")
	}
}

func TestEvalExists_Missing(t *testing.T) {
	cond := WatchCondition{
		Type:     CondExists,
		JSONPath: "pid",
	}
	output := map[string]interface{}{"name": "test"}

	result, err := Evaluate(cond, output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Triggered {
		t.Error("expected triggered when value is missing but should exist")
	}
}

func TestEvalCustom_EQ(t *testing.T) {
	cond := WatchCondition{
		Type:      CondCustom,
		JSONPath:  "enabled",
		Operator:  OpEQ,
		Threshold: "true",
	}
	output := map[string]interface{}{"enabled": "true"}

	result, err := Evaluate(cond, output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Triggered {
		t.Error("expected triggered for custom eq match")
	}
}

func TestEvalCustom_NE(t *testing.T) {
	cond := WatchCondition{
		Type:      CondCustom,
		JSONPath:  "enabled",
		Operator:  OpNE,
		Threshold: "true",
	}
	output := map[string]interface{}{"enabled": "false"}

	result, err := Evaluate(cond, output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Triggered {
		t.Error("expected triggered for custom ne match")
	}
}

func TestEvalUnsupportedType(t *testing.T) {
	cond := WatchCondition{Type: "unknown_type"}
	output := map[string]interface{}{}

	_, err := Evaluate(cond, output)
	if err == nil {
		t.Error("expected error for unsupported condition type")
	}
}

func TestExtractByJSONPath_DotPrefix(t *testing.T) {
	data := map[string]interface{}{
		"a": map[string]interface{}{
			"b": "value",
		},
	}

	val, err := extractByJSONPath("$.a.b", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "value" {
		t.Errorf("expected 'value', got %v", val)
	}
}

func TestToFloat64_Various(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected float64
		hasErr   bool
	}{
		{float64(1.5), 1.5, false},
		{int(42), 42.0, false},
		{int64(100), 100.0, false},
		{"3.14", 3.14, false},
		{"not_a_number", 0, true},
		{true, 0, true},
	}

	for _, tt := range tests {
		result, err := toFloat64(tt.input)
		if tt.hasErr && err == nil {
			t.Errorf("expected error for input %v", tt.input)
		}
		if !tt.hasErr && err != nil {
			t.Errorf("unexpected error for input %v: %v", tt.input, err)
		}
		if !tt.hasErr && result != tt.expected {
			t.Errorf("expected %v, got %v for input %v", tt.expected, result, tt.input)
		}
	}
}
