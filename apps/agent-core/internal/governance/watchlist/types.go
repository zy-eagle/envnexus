package watchlist

import "time"

type WatchItemSource string

const (
	SourceUser         WatchItemSource = "user"
	SourceLLMSuggested WatchItemSource = "llm_suggested"
	SourceBuiltin      WatchItemSource = "builtin"
	SourcePlatform     WatchItemSource = "platform"
)

type ConditionType string

const (
	CondThreshold  ConditionType = "threshold"
	CondExists     ConditionType = "exists"
	CondReachable  ConditionType = "reachable"
	CondContains   ConditionType = "contains"
	CondCustom     ConditionType = "custom"
)

type Operator string

const (
	OpLT          Operator = "lt"
	OpGT          Operator = "gt"
	OpEQ          Operator = "eq"
	OpNE          Operator = "ne"
	OpContains    Operator = "contains"
	OpNotContains Operator = "not_contains"
)

type AlertSeverity string

const (
	SeverityInfo     AlertSeverity = "info"
	SeverityWarning  AlertSeverity = "warning"
	SeverityCritical AlertSeverity = "critical"
)

type WatchItem struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	Source          WatchItemSource        `json:"source"`
	ToolName        string                 `json:"tool_name"`
	ToolParams      map[string]interface{} `json:"tool_params,omitempty"`
	Condition       WatchCondition         `json:"condition"`
	Interval        time.Duration          `json:"interval"`
	Enabled         bool                   `json:"enabled"`
	LastCheckAt     time.Time              `json:"last_check_at,omitempty"`
	LastStatus      string                 `json:"last_status,omitempty"`
	ConsecutiveFail int                    `json:"consecutive_fail"`
	CreatedAt       time.Time              `json:"created_at"`
}

type WatchCondition struct {
	Type      ConditionType `json:"type"`
	JSONPath  string        `json:"json_path,omitempty"`
	Operator  Operator      `json:"operator,omitempty"`
	Threshold interface{}   `json:"threshold,omitempty"`
	Pattern   string        `json:"pattern,omitempty"`
}

type WatchAlert struct {
	ID          string        `json:"id"`
	WatchItemID string        `json:"watch_item_id"`
	ItemName    string        `json:"item_name"`
	Severity    AlertSeverity `json:"severity"`
	Message     string        `json:"message"`
	Value       interface{}   `json:"value,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
	Resolved    bool          `json:"resolved"`
}

type HealthScore struct {
	Score       int    `json:"score"`
	Total       int    `json:"total_items"`
	Healthy     int    `json:"healthy"`
	Warning     int    `json:"warning"`
	Critical    int    `json:"critical"`
	LastUpdated string `json:"last_updated"`
}
