package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strings"
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
		"detail":  fmt.Sprintf("Reached maximum of %d tool iterations. Generating summary...", l.maxIterations),
		"warning": "max_iterations_reached",
	}})

	fullMessages = append(fullMessages, router.Message{
		Role:    "user",
		Content: "You have reached the maximum number of tool call iterations. Stop calling tools. Summarize what you accomplished, what failed (include specific error messages), and what remains to be done.",
	})

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
		errMsg := fmt.Sprintf("tool '%s' not found", toolName)
		result := map[string]interface{}{
			"error": errMsg,
		}
		l.emit(Event{Type: EventToolResult, Content: map[string]interface{}{
			"tool_name": toolName,
			"call_id":   tc.ID,
			"status":    "failed",
			"error":     errMsg,
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
			"error":       err.Error(),
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

	evtContent := map[string]interface{}{
		"tool_name":   toolName,
		"call_id":     tc.ID,
		"status":      toolResult.Status,
		"summary":     toolResult.Summary,
		"duration_ms": toolResult.DurationMs,
	}
	if toolResult.Error != "" {
		evtContent["error"] = toolResult.Error
	}
	if toolResult.Status == "failed" {
		evtContent["output"] = toolResult.Output
	}
	l.emit(Event{Type: EventToolResult, Content: evtContent})

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
	envInfo := collectEnvironmentInfo()
	return fmt.Sprintf(`You are EnvNexus Agent, a local IT diagnostic and operations assistant running on the user's machine.

%s

You have access to a set of tools that can inspect and operate on this machine. Use them to answer the user's questions accurately.

Guidelines:
- When the user asks about system state (IP, files, processes, etc.), call the appropriate tool to get real data. Do NOT guess or make up information.
- You may call multiple tools in sequence if needed to fully answer a question.
- Present results in a clear, readable format. Summarize key findings.
- ALWAYS call tools directly when the user requests an action. NEVER ask the user for confirmation yourself — the system has a built-in approval mechanism that will automatically prompt the user for approval on sensitive operations. Just call the tool and the system handles the rest.
- Do NOT simulate or role-play an approval process in your text responses. Do NOT say things like "Do you want me to proceed?" or "Please confirm". Simply call the appropriate tool.
- If a tool call fails, explain the error and suggest alternatives.
- When using shell_exec, generate commands appropriate for the shell listed above. On Windows use PowerShell syntax; on Linux/macOS use sh/bash syntax.
- Respond in the same language as the user's message.`, envInfo)
}

func collectEnvironmentInfo() string {
	hostname, _ := os.Hostname()

	var lines []string
	lines = append(lines, fmt.Sprintf("OS: %s", runtime.GOOS))
	lines = append(lines, fmt.Sprintf("Architecture: %s", runtime.GOARCH))
	if hostname != "" {
		lines = append(lines, fmt.Sprintf("Hostname: %s", hostname))
	}

	switch runtime.GOOS {
	case "windows":
		if v := os.Getenv("OS"); v != "" {
			lines = append(lines, fmt.Sprintf("OS Version: %s", v))
		}
		if v := os.Getenv("PROCESSOR_ARCHITECTURE"); v != "" {
			lines = append(lines, fmt.Sprintf("Processor: %s", v))
		}
		lines = append(lines, "Shell (shell_exec uses): PowerShell (-NoProfile -NonInteractive)")
		lines = append(lines, "Use PowerShell syntax: cmdlets (Get-ChildItem, Rename-Item, Get-Process), pipelines, variables ($env:PATH). Avoid cmd-only syntax like 'dir /s' — use 'Get-ChildItem -Recurse' instead.")
		if v := os.Getenv("USERPROFILE"); v != "" {
			lines = append(lines, fmt.Sprintf("Home: %s", v))
		}
		if v := os.Getenv("SystemRoot"); v != "" {
			lines = append(lines, fmt.Sprintf("SystemRoot: %s", v))
		}
	default:
		lines = append(lines, "Shell (shell_exec uses): sh -c")
		if userShell := os.Getenv("SHELL"); userShell != "" {
			lines = append(lines, fmt.Sprintf("User default shell: %s", userShell))
		}
		if v := os.Getenv("HOME"); v != "" {
			lines = append(lines, fmt.Sprintf("Home: %s", v))
		}
		if v := os.Getenv("LANG"); v != "" {
			lines = append(lines, fmt.Sprintf("Locale: %s", v))
		}
	}

	if v := os.Getenv("PATH"); v != "" {
		sep := ":"
		if runtime.GOOS == "windows" {
			sep = ";"
		}
		paths := strings.Split(v, sep)
		if len(paths) > 10 {
			paths = paths[:10]
		}
		lines = append(lines, fmt.Sprintf("PATH (first %d): %s", len(paths), strings.Join(paths, sep)))
	}

	if wd, err := os.Getwd(); err == nil {
		lines = append(lines, fmt.Sprintf("Working Directory: %s", wd))
	}

	return "System Environment:\n" + strings.Join(lines, "\n")
}
