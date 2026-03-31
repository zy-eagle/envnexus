package tools

import (
	"context"
	"sort"
)

type ToolResult struct {
	ToolName   string      `json:"tool_name"`
	Status     string      `json:"status"`
	Summary    string      `json:"summary"`
	Output     interface{} `json:"output"`
	Error      string      `json:"error,omitempty"`
	DurationMs int64       `json:"duration_ms"`
}

type ParamProperty struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Enum        []string `json:"enum,omitempty"`
}

type ParamSchema struct {
	Type       string                   `json:"type"`
	Properties map[string]ParamProperty `json:"properties,omitempty"`
	Required   []string                 `json:"required,omitempty"`
}

func NoParams() *ParamSchema {
	return &ParamSchema{Type: "object", Properties: map[string]ParamProperty{}}
}

type Tool interface {
	Name() string
	Description() string
	IsReadOnly() bool
	RiskLevel() string
	Parameters() *ParamSchema
	Execute(ctx context.Context, params map[string]interface{}) (*ToolResult, error)
}

// ApprovalChecker is an optional interface that tools can implement
// to provide per-invocation approval decisions based on actual parameters.
// If a tool implements this, the agent loop calls NeedsApproval before execution.
// If not implemented, the loop falls back to !IsReadOnly().
type ApprovalChecker interface {
	NeedsApproval(params map[string]interface{}) bool
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

type OpenAIFunctionDef struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  *ParamSchema `json:"parameters"`
}

type OpenAIToolDef struct {
	Type     string            `json:"type"`
	Function OpenAIFunctionDef `json:"function"`
}

func (r *Registry) ToOpenAITools() []OpenAIToolDef {
	list := r.List()
	sort.Slice(list, func(i, j int) bool { return list[i].Name() < list[j].Name() })

	defs := make([]OpenAIToolDef, 0, len(list))
	for _, t := range list {
		schema := t.Parameters()
		if schema == nil {
			schema = NoParams()
		}
		defs = append(defs, OpenAIToolDef{
			Type: "function",
			Function: OpenAIFunctionDef{
				Name:        t.Name(),
				Description: t.Description(),
				Parameters:  schema,
			},
		})
	}
	return defs
}
