package tools

import "context"

// ToolResult represents the structured output of a tool execution.
type ToolResult struct {
	ToolName   string      `json:"tool_name"`
	Status     string      `json:"status"` // "succeeded", "failed"
	Summary    string      `json:"summary"`
	Output     interface{} `json:"output"`
	Error      string      `json:"error,omitempty"`
	DurationMs int64       `json:"duration_ms"`
}

// Tool defines the interface that all diagnostic and repair tools must implement.
type Tool interface {
	Name() string
	Description() string
	IsReadOnly() bool
	Execute(ctx context.Context, params map[string]interface{}) (*ToolResult, error)
}

// Registry manages available tools.
type Registry struct {
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

func (r *Registry) Register(t Tool) {
	r.tools[t.Name()] = t
}

func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}
