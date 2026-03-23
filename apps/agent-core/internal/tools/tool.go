package tools

import "context"

type ToolResult struct {
	ToolName   string      `json:"tool_name"`
	Status     string      `json:"status"`
	Summary    string      `json:"summary"`
	Output     interface{} `json:"output"`
	Error      string      `json:"error,omitempty"`
	DurationMs int64       `json:"duration_ms"`
}

type Tool interface {
	Name() string
	Description() string
	IsReadOnly() bool
	RiskLevel() string
	Execute(ctx context.Context, params map[string]interface{}) (*ToolResult, error)
}

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

func (r *Registry) Count() int {
	return len(r.tools)
}

func (r *Registry) List() []Tool {
	list := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		list = append(list, t)
	}
	return list
}
