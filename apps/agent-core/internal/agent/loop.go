package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"runtime"
	"time"

	"github.com/zy-eagle/envnexus/apps/agent-core/internal/llm/router"
	"github.com/zy-eagle/envnexus/apps/agent-core/internal/tools"
)

const (
	DefaultMaxIterations = 10
	DefaultMaxTokens     = 4096
	DefaultTemperature   = 0.3
)

type ApprovalRequest struct {
	ToolName    string                 `json:"tool_name"`
	Description string                 `json:"description"`
	RiskLevel   string                 `json:"risk_level"`
	Params      map[string]interface{} `json:"params"`
}

type ApprovalResponse struct {
	Approved bool `json:"approved"`
}

type EventType string

const (
	EventThinking         EventType = "thinking"
	EventToolStart        EventType = "tool_start"
	EventToolResult       EventType = "tool_result"
	EventApprovalRequired EventType = "approval_required"
	EventMessage          EventType = "message"
	EventError            EventType = "error"
)

type Event struct {
	Type    EventType   `json:"type"`
	Content interface{} `json:"content"`
}

type EventHandler func(event Event)

type ApprovalHandler func(req ApprovalRequest) ApprovalResponse

type Loop struct {
	llmRouter       *router.Router
	registry        *tools.Registry
	maxIterations   int
	onEvent         EventHandler
	onApproval      ApprovalHandler
}

type LoopOption func(*Loop)

func WithMaxIterations(n int) LoopOption {
	return func(l *Loop) { l.maxIterations = n }
}

func WithEventHandler(h EventHandler) LoopOption {
	return func(l *Loop) { l.onEvent = h }
}

func WithApprovalHandler(h ApprovalHandler) LoopOption {
	return func(l *Loop) { l.onApproval = h }
}

func NewLoop(llmRouter *router.Router, registry *tools.Registry, opts ...LoopOption) *Loop {
	l := &Loop{
		llmRouter:     llmRouter,
		registry:      registry,
		maxIterations: DefaultMaxIterations,
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

func (l *Loop) emit(evt Event) {
	if l.onEvent != nil {
		l.onEvent(evt)
	}
}

func (l *Loop) Run(ctx context.Context, messages []router.Message) (string, error) {
	toolDefs := l.buildToolDefinitions()
	systemPrompt := l.buildSystemPrompt()

	fullMessages := make([]router.Message, 0, len(messages)+1)
	fullMessages = append(fullMessages, router.Message{
		Role:    "system",
		Content: systemPrompt,
	})
	fullMessages = append(fullMessages, messages...)

	for i := 0; i < l.maxIterations; i++ {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}

		slog.Info("[agent] Loop iteration", "iteration", i+1, "messages", len(fullMessages))

		l.emit(Event{Type: EventThinking, Content: map[string]interface{}{
			"iteration": i + 1,
			"detail":    "Thinking...",
		}})

		resp, err := l.llmRouter.Complete(ctx, &router.CompletionRequest{
			Messages:    fullMessages,
			Tools:       toolDefs,
			MaxTokens:   DefaultMaxTokens,
			Temperature: DefaultTemperature,
		})
		if err != nil {
			return "", fmt.Errorf("llm completion failed at iteration %d: %w", i+1, err)
		}

		if len(resp.ToolCalls) == 0 {
			l.emit(Event{Type: EventMessage, Content: map[string]interface{}{
				"content": resp.Content,
			}})
			return resp.Content, nil
		}

		assistantMsg := router.Message{
			Role:      "assistant",
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		}
		fullMessages = append(fullMessages, assistantMsg)

		for _, tc := range resp.ToolCalls {
			if ctx.Err() != nil {
				return "", ctx.Err()
			}
			toolResult := l.executeTool(ctx, tc)

			resultJSON, _ := json.Marshal(toolResult)
			fullMessages = append(fullMessages, router.Message{
				Role:       "tool",
				Content:    string(resultJSON),
				ToolCallID: tc.ID,
				Name:       tc.Function.Name,
			})
		}
	}

	l.emit(Event{Type: EventThinking, Content: map[string]interface{}{
		"detail": "Max iterations reached, generating final response...",
	}})

	resp, err := l.llmRouter.Complete(ctx, &router.CompletionRequest{
		Messages:    fullMessages,
		MaxTokens:   DefaultMaxTokens,
		Temperature: DefaultTemperature,
	})
	if err != nil {
		return "", fmt.Errorf("final completion failed: %w", err)
	}

	l.emit(Event{Type: EventMessage, Content: map[string]interface{}{
		"content": resp.Content,
	}})
	return resp.Content, nil
}

func (l *Loop) executeTool(ctx context.Context, tc router.ToolCall) map[string]interface{} {
	toolName := tc.Function.Name

	l.emit(Event{Type: EventToolStart, Content: map[string]interface{}{
		"tool_name": toolName,
		"arguments": tc.Function.Arguments,
		"call_id":   tc.ID,
	}})

	tool, found := l.registry.Get(toolName)
	if !found {
		result := map[string]interface{}{
			"error": fmt.Sprintf("tool '%s' not found", toolName),
		}
		l.emit(Event{Type: EventToolResult, Content: map[string]interface{}{
			"tool_name": toolName,
			"call_id":   tc.ID,
			"status":    "failed",
			"output":    result,
		}})
		return result
	}

	needsApproval := !tool.IsReadOnly()
	if checker, ok := tool.(tools.ApprovalChecker); ok {
		params := make(map[string]interface{})
		if tc.Function.Arguments != "" {
			_ = json.Unmarshal([]byte(tc.Function.Arguments), &params)
		}
		needsApproval = checker.NeedsApproval(params)
	}

	if needsApproval {
		approved := l.requestApproval(tool, tc)
		if !approved {
			result := map[string]interface{}{
				"status":  "rejected",
				"message": "User rejected this operation",
			}
			l.emit(Event{Type: EventToolResult, Content: map[string]interface{}{
				"tool_name": toolName,
				"call_id":   tc.ID,
				"status":    "rejected",
				"output":    result,
			}})
			return result
		}
	}

	params := make(map[string]interface{})
	if tc.Function.Arguments != "" {
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &params); err != nil {
			slog.Warn("[agent] Failed to parse tool arguments", "tool", toolName, "error", err)
		}
	}

	start := time.Now()
	toolResult, err := tool.Execute(ctx, params)
	elapsed := time.Since(start)

	if err != nil {
		result := map[string]interface{}{
			"error":       err.Error(),
			"duration_ms": elapsed.Milliseconds(),
		}
		l.emit(Event{Type: EventToolResult, Content: map[string]interface{}{
			"tool_name":   toolName,
			"call_id":     tc.ID,
			"status":      "failed",
			"output":      result,
			"duration_ms": elapsed.Milliseconds(),
		}})
		return result
	}

	result := map[string]interface{}{
		"status":      toolResult.Status,
		"summary":     toolResult.Summary,
		"output":      toolResult.Output,
		"duration_ms": toolResult.DurationMs,
	}
	if toolResult.Error != "" {
		result["error"] = toolResult.Error
	}

	l.emit(Event{Type: EventToolResult, Content: map[string]interface{}{
		"tool_name":   toolName,
		"call_id":     tc.ID,
		"status":      toolResult.Status,
		"summary":     toolResult.Summary,
		"duration_ms": toolResult.DurationMs,
	}})

	slog.Info("[agent] Tool executed", "tool", toolName, "status", toolResult.Status, "duration_ms", toolResult.DurationMs)
	return result
}

func (l *Loop) requestApproval(tool tools.Tool, tc router.ToolCall) bool {
	if l.onApproval == nil {
		slog.Warn("[agent] Write tool requested but no approval handler set, denying", "tool", tool.Name())
		return false
	}

	var params map[string]interface{}
	if tc.Function.Arguments != "" {
		json.Unmarshal([]byte(tc.Function.Arguments), &params)
	}

	req := ApprovalRequest{
		ToolName:    tool.Name(),
		Description: tool.Description(),
		RiskLevel:   tool.RiskLevel(),
		Params:      params,
	}

	resp := l.onApproval(req)
	return resp.Approved
}

func (l *Loop) buildToolDefinitions() []router.ToolDefinition {
	openaiTools := l.registry.ToOpenAITools()
	defs := make([]router.ToolDefinition, len(openaiTools))
	for i, t := range openaiTools {
		defs[i] = router.ToolDefinition{
			Type: t.Type,
			Function: router.FunctionDef{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				Parameters:  t.Function.Parameters,
			},
		}
	}
	return defs
}

func (l *Loop) buildSystemPrompt() string {
	return fmt.Sprintf(`You are EnvNexus Agent, a local IT diagnostic and operations assistant running on the user's machine.

System: %s/%s

You have access to a set of tools that can inspect and operate on this machine. Use them to answer the user's questions accurately.

Guidelines:
- When the user asks about system state (IP, files, processes, etc.), call the appropriate tool to get real data. Do NOT guess or make up information.
- You may call multiple tools in sequence if needed to fully answer a question.
- Present results in a clear, readable format. Summarize key findings.
- For write operations (service restart, config changes, etc.), explain what you plan to do before calling the tool.
- If a tool call fails, explain the error and suggest alternatives.
- Respond in the same language as the user's message.`, runtime.GOOS, runtime.GOARCH)
}
