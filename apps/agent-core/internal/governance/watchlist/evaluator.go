package watchlist

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type EvalResult struct {
	Triggered bool        `json:"triggered"`
	Value     interface{} `json:"value,omitempty"`
	Message   string      `json:"message,omitempty"`
}

func Evaluate(cond WatchCondition, toolOutput map[string]interface{}) (*EvalResult, error) {
	switch cond.Type {
	case CondThreshold:
		return evalThreshold(cond, toolOutput)
	case CondExists:
		return evalExists(cond, toolOutput)
	case CondReachable:
		return evalReachable(toolOutput)
	case CondContains:
		return evalContains(cond, toolOutput)
	case CondCustom:
		return evalCustom(cond, toolOutput)
	default:
		return nil, fmt.Errorf("unsupported condition type: %s", cond.Type)
	}
}

func evalThreshold(cond WatchCondition, output map[string]interface{}) (*EvalResult, error) {
	raw, err := extractByJSONPath(cond.JSONPath, output)
	if err != nil {
		return &EvalResult{Triggered: true, Message: fmt.Sprintf("failed to extract value at %s: %v", cond.JSONPath, err)}, nil
	}

	actual, err := toFloat64(raw)
	if err != nil {
		return &EvalResult{Triggered: true, Value: raw, Message: fmt.Sprintf("value at %s is not numeric: %v", cond.JSONPath, raw)}, nil
	}

	threshold, err := toFloat64(cond.Threshold)
	if err != nil {
		return nil, fmt.Errorf("threshold is not numeric: %v", cond.Threshold)
	}

	triggered := false
	switch cond.Operator {
	case OpLT:
		triggered = actual < threshold
	case OpGT:
		triggered = actual > threshold
	case OpEQ:
		triggered = actual == threshold
	case OpNE:
		triggered = actual != threshold
	default:
		return nil, fmt.Errorf("unsupported operator for threshold: %s", cond.Operator)
	}

	result := &EvalResult{Triggered: triggered, Value: actual}
	if triggered {
		result.Message = fmt.Sprintf("value %.2f %s threshold %.2f", actual, cond.Operator, threshold)
	}
	return result, nil
}

func evalExists(cond WatchCondition, output map[string]interface{}) (*EvalResult, error) {
	raw, err := extractByJSONPath(cond.JSONPath, output)
	exists := err == nil && raw != nil

	shouldExist := true
	if cond.Operator == OpEQ {
		if b, ok := cond.Threshold.(bool); ok {
			shouldExist = b
		}
	}

	triggered := exists != shouldExist
	result := &EvalResult{Triggered: triggered, Value: exists}
	if triggered {
		if shouldExist {
			result.Message = fmt.Sprintf("expected value at %s to exist", cond.JSONPath)
		} else {
			result.Message = fmt.Sprintf("expected value at %s to not exist", cond.JSONPath)
		}
	}
	return result, nil
}

func evalReachable(output map[string]interface{}) (*EvalResult, error) {
	status, _ := output["status"].(string)
	errMsg, _ := output["error"].(string)

	triggered := status != "success" && status != "ok"
	result := &EvalResult{Triggered: triggered, Value: status}
	if triggered {
		result.Message = "target is unreachable"
		if errMsg != "" {
			result.Message += ": " + errMsg
		}
	}
	return result, nil
}

func evalContains(cond WatchCondition, output map[string]interface{}) (*EvalResult, error) {
	raw, err := extractByJSONPath(cond.JSONPath, output)
	if err != nil {
		return &EvalResult{Triggered: true, Message: fmt.Sprintf("failed to extract value at %s: %v", cond.JSONPath, err)}, nil
	}

	str := fmt.Sprintf("%v", raw)
	pattern := cond.Pattern
	if pattern == "" {
		if s, ok := cond.Threshold.(string); ok {
			pattern = s
		}
	}

	var triggered bool
	switch cond.Operator {
	case OpContains, "":
		triggered = strings.Contains(str, pattern)
	case OpNotContains:
		triggered = !strings.Contains(str, pattern)
	default:
		return nil, fmt.Errorf("unsupported operator for contains: %s", cond.Operator)
	}

	result := &EvalResult{Triggered: triggered, Value: str}
	if triggered {
		result.Message = fmt.Sprintf("value contains pattern %q", pattern)
		if cond.Operator == OpNotContains {
			result.Message = fmt.Sprintf("value does not contain pattern %q", pattern)
		}
	}
	return result, nil
}

func evalCustom(cond WatchCondition, output map[string]interface{}) (*EvalResult, error) {
	raw, err := extractByJSONPath(cond.JSONPath, output)
	if err != nil {
		return &EvalResult{Triggered: true, Message: fmt.Sprintf("failed to extract value at %s: %v", cond.JSONPath, err)}, nil
	}

	actualStr := fmt.Sprintf("%v", raw)
	thresholdStr := fmt.Sprintf("%v", cond.Threshold)

	var triggered bool
	switch cond.Operator {
	case OpEQ:
		triggered = actualStr == thresholdStr
	case OpNE:
		triggered = actualStr != thresholdStr
	case OpContains:
		triggered = strings.Contains(actualStr, thresholdStr)
	case OpNotContains:
		triggered = !strings.Contains(actualStr, thresholdStr)
	case OpLT, OpGT:
		a, errA := toFloat64(raw)
		b, errB := toFloat64(cond.Threshold)
		if errA != nil || errB != nil {
			return &EvalResult{Triggered: true, Message: "custom condition requires numeric values for lt/gt"}, nil
		}
		if cond.Operator == OpLT {
			triggered = a < b
		} else {
			triggered = a > b
		}
	default:
		return nil, fmt.Errorf("unsupported operator for custom: %s", cond.Operator)
	}

	result := &EvalResult{Triggered: triggered, Value: raw}
	if triggered {
		result.Message = fmt.Sprintf("custom condition met: %v %s %v", raw, cond.Operator, cond.Threshold)
	}
	return result, nil
}

// extractByJSONPath does a simple dot-separated path lookup on a nested map.
// Example: "output.cpu_percent" extracts output["cpu_percent"] from the top-level map.
func extractByJSONPath(path string, data map[string]interface{}) (interface{}, error) {
	if path == "" || path == "." {
		return data, nil
	}

	path = strings.TrimPrefix(path, "$.")
	parts := strings.Split(path, ".")

	var current interface{} = data
	for _, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			raw, err := tryParseJSON(current)
			if err != nil {
				return nil, fmt.Errorf("cannot traverse into non-object at %q", part)
			}
			m = raw
		}
		val, exists := m[part]
		if !exists {
			return nil, fmt.Errorf("key %q not found", part)
		}
		current = val
	}
	return current, nil
}

func tryParseJSON(v interface{}) (map[string]interface{}, error) {
	s, ok := v.(string)
	if !ok {
		return nil, fmt.Errorf("not a string")
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil, err
	}
	return m, nil
}

func toFloat64(v interface{}) (float64, error) {
	switch n := v.(type) {
	case float64:
		return n, nil
	case float32:
		return float64(n), nil
	case int:
		return float64(n), nil
	case int64:
		return float64(n), nil
	case int32:
		return float64(n), nil
	case json.Number:
		return n.Float64()
	case string:
		return strconv.ParseFloat(n, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", v)
	}
}
