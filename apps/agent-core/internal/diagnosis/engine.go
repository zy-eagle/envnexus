package diagnosis

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/zy-eagle/envnexus/apps/agent-core/internal/llm/router"
	"github.com/zy-eagle/envnexus/apps/agent-core/internal/tools"
)

type DiagnosisPlan struct {
	ProblemType string   `json:"problem_type"`
	Scope       string   `json:"scope"`
	RiskBias    string   `json:"risk_bias"`
	ToolNames   []string `json:"tool_names"`
}

type DiagnosisResult struct {
	ProblemType        string           `json:"problem_type"`
	Confidence         float64          `json:"confidence"`
	Findings           []Finding        `json:"findings"`
	RecommendedActions []ActionDraft    `json:"recommended_actions"`
	ApprovalRequired   bool             `json:"approval_required"`
	NextStep           string           `json:"next_step"`
	Evidence           map[string]interface{} `json:"evidence"`
	DurationMs         int64            `json:"duration_ms"`
}

type Finding struct {
	Source  string `json:"source"`
	Summary string `json:"summary"`
	Level   string `json:"level"`
}

type ActionDraft struct {
	ToolName    string                 `json:"tool_name"`
	Description string                 `json:"description"`
	RiskLevel   string                 `json:"risk_level"`
	Params      map[string]interface{} `json:"params"`
}

type Engine struct {
	registry  *tools.Registry
	llmRouter *router.Router
}

func NewEngine(registry *tools.Registry, llmRouter *router.Router) *Engine {
	return &Engine{
		registry:  registry,
		llmRouter: llmRouter,
	}
}

func (e *Engine) Plan(ctx context.Context, sessionID, input string) (*DiagnosisPlan, error) {
	log.Printf("[diagnosis] Planning for session %s: %s", sessionID, input)

	plan, err := e.stepIntentParse(ctx, input)
	if err != nil {
		log.Printf("[diagnosis] IntentParse failed, using heuristic: %v", err)
		plan = e.heuristicPlan(input)
	}

	plan.ToolNames = e.stepEvidencePlan(plan)
	log.Printf("[diagnosis] Plan: type=%s, tools=%v", plan.ProblemType, plan.ToolNames)
	return plan, nil
}

func (e *Engine) Execute(ctx context.Context, sessionID string, plan *DiagnosisPlan) (*DiagnosisResult, error) {
	start := time.Now()
	log.Printf("[diagnosis] Executing plan for session %s (type=%s)", sessionID, plan.ProblemType)

	evidence := e.stepEvidenceCollect(ctx, plan.ToolNames)

	reasoning, err := e.stepReasoning(ctx, plan, evidence)
	if err != nil {
		log.Printf("[diagnosis] Reasoning via LLM failed, using local analysis: %v", err)
		reasoning = e.localReasoning(plan, evidence)
	}

	reasoning.Evidence = evidence
	reasoning.DurationMs = time.Since(start).Milliseconds()

	if len(reasoning.RecommendedActions) > 0 {
		e.stepActionDraft(reasoning)
	}

	log.Printf("[diagnosis] Complete: type=%s confidence=%.2f findings=%d actions=%d duration=%dms",
		reasoning.ProblemType, reasoning.Confidence, len(reasoning.Findings),
		len(reasoning.RecommendedActions), reasoning.DurationMs)

	return reasoning, nil
}

func (e *Engine) RunDiagnosis(ctx context.Context, sessionID, intent string) (*DiagnosisResult, error) {
	plan, err := e.Plan(ctx, sessionID, intent)
	if err != nil {
		return nil, fmt.Errorf("plan: %w", err)
	}
	return e.Execute(ctx, sessionID, plan)
}

// Step 1: IntentParse — use LLM to classify the problem
func (e *Engine) stepIntentParse(ctx context.Context, input string) (*DiagnosisPlan, error) {
	if e.llmRouter == nil {
		return nil, fmt.Errorf("no LLM router configured")
	}

	prompt := fmt.Sprintf(`You are an IT diagnostic assistant. Analyze the user's problem description and return a JSON object with:
- "problem_type": one of "network", "dns", "service", "performance", "disk", "auth", "general"
- "scope": one of "local", "network", "system"
- "risk_bias": one of "conservative", "moderate", "aggressive"

User's problem: %s

Respond ONLY with the JSON object, no other text.`, input)

	resp, err := e.llmRouter.Complete(ctx, &router.CompletionRequest{
		Messages: []router.Message{
			{Role: "system", Content: "You are a structured diagnostic classifier. Output only valid JSON."},
			{Role: "user", Content: prompt},
		},
		MaxTokens:   256,
		Temperature: 0.1,
	})
	if err != nil {
		return nil, err
	}

	var plan DiagnosisPlan
	content := strings.TrimSpace(resp.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	if err := json.Unmarshal([]byte(content), &plan); err != nil {
		return nil, fmt.Errorf("parse LLM response: %w (content: %s)", err, content)
	}
	return &plan, nil
}

// Step 2: EvidencePlan — select read-only tools based on problem type
func (e *Engine) stepEvidencePlan(plan *DiagnosisPlan) []string {
	toolMapping := map[string][]string{
		"network":     {"read_network_config", "read_system_info"},
		"dns":         {"read_network_config", "read_system_info"},
		"service":     {"read_system_info"},
		"performance": {"read_system_info"},
		"disk":        {"read_system_info"},
		"auth":        {"read_system_info"},
		"general":     {"read_system_info", "read_network_config"},
	}

	candidates := toolMapping[plan.ProblemType]
	if len(candidates) == 0 {
		candidates = toolMapping["general"]
	}

	var available []string
	for _, name := range candidates {
		if t, ok := e.registry.Get(name); ok && t.IsReadOnly() {
			available = append(available, name)
		}
	}
	return available
}

// Step 3: EvidenceCollect — execute read-only tools in parallel
func (e *Engine) stepEvidenceCollect(ctx context.Context, toolNames []string) map[string]interface{} {
	evidence := make(map[string]interface{})
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, name := range toolNames {
		t, ok := e.registry.Get(name)
		if !ok {
			continue
		}
		wg.Add(1)
		go func(tool tools.Tool, toolName string) {
			defer wg.Done()
			result, err := tool.Execute(ctx, nil)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				evidence[toolName] = map[string]interface{}{"error": err.Error()}
				log.Printf("[diagnosis] Tool %s failed: %v", toolName, err)
			} else {
				evidence[toolName] = result.Output
				log.Printf("[diagnosis] Tool %s collected: %s", toolName, result.Summary)
			}
		}(t, name)
	}
	wg.Wait()
	return evidence
}

// Step 4: Reasoning — use LLM to analyze evidence and generate diagnosis
func (e *Engine) stepReasoning(ctx context.Context, plan *DiagnosisPlan, evidence map[string]interface{}) (*DiagnosisResult, error) {
	if e.llmRouter == nil {
		return nil, fmt.Errorf("no LLM router configured")
	}

	evidenceJSON, _ := json.MarshalIndent(evidence, "", "  ")

	prompt := fmt.Sprintf(`You are an IT diagnostic engine. Given the problem type and collected evidence, produce a diagnosis.

Problem type: %s
Scope: %s

Evidence collected:
%s

Return a JSON object with:
- "problem_type": string
- "confidence": float between 0.0 and 1.0
- "findings": array of {"source": string, "summary": string, "level": "info"|"warning"|"error"}
- "recommended_actions": array of {"tool_name": string, "description": string, "risk_level": "L0"|"L1"|"L2"|"L3", "params": {}}
- "approval_required": boolean (true if any action is L1 or above)
- "next_step": string describing what to do next

Respond ONLY with the JSON object.`, plan.ProblemType, plan.Scope, string(evidenceJSON))

	resp, err := e.llmRouter.Complete(ctx, &router.CompletionRequest{
		Messages: []router.Message{
			{Role: "system", Content: "You are a structured IT diagnostic engine. Output only valid JSON."},
			{Role: "user", Content: prompt},
		},
		MaxTokens:   1024,
		Temperature: 0.2,
	})
	if err != nil {
		return nil, err
	}

	var result DiagnosisResult
	content := strings.TrimSpace(resp.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("parse reasoning response: %w", err)
	}
	return &result, nil
}

// Step 5: ActionDraft — validate and enrich recommended actions
func (e *Engine) stepActionDraft(result *DiagnosisResult) {
	var validated []ActionDraft
	for _, action := range result.RecommendedActions {
		t, ok := e.registry.Get(action.ToolName)
		if !ok {
			log.Printf("[diagnosis] ActionDraft: tool %s not found, skipping", action.ToolName)
			continue
		}
		action.RiskLevel = t.RiskLevel()
		if action.RiskLevel != "L0" {
			result.ApprovalRequired = true
		}
		validated = append(validated, action)
	}
	result.RecommendedActions = validated
}

func (e *Engine) heuristicPlan(input string) *DiagnosisPlan {
	lower := strings.ToLower(input)
	plan := &DiagnosisPlan{
		ProblemType: "general",
		Scope:       "local",
		RiskBias:    "conservative",
	}
	switch {
	case strings.Contains(lower, "dns") || strings.Contains(lower, "resolve") || strings.Contains(lower, "域名"):
		plan.ProblemType = "dns"
		plan.Scope = "network"
	case strings.Contains(lower, "network") || strings.Contains(lower, "网络") || strings.Contains(lower, "connect"):
		plan.ProblemType = "network"
		plan.Scope = "network"
	case strings.Contains(lower, "service") || strings.Contains(lower, "服务") || strings.Contains(lower, "restart"):
		plan.ProblemType = "service"
	case strings.Contains(lower, "slow") || strings.Contains(lower, "慢") || strings.Contains(lower, "performance"):
		plan.ProblemType = "performance"
	case strings.Contains(lower, "disk") || strings.Contains(lower, "磁盘") || strings.Contains(lower, "space"):
		plan.ProblemType = "disk"
	}
	return plan
}

func (e *Engine) localReasoning(plan *DiagnosisPlan, evidence map[string]interface{}) *DiagnosisResult {
	findings := make([]Finding, 0)
	for source, data := range evidence {
		findings = append(findings, Finding{
			Source:  source,
			Summary: fmt.Sprintf("Data collected from %s", source),
			Level:   "info",
		})
		_ = data
	}

	var actions []ActionDraft
	switch plan.ProblemType {
	case "dns":
		actions = append(actions, ActionDraft{
			ToolName:    "dns.flush_cache",
			Description: "Flush DNS cache to resolve stale entries",
			RiskLevel:   "L2",
			Params:      map[string]interface{}{},
		})
	case "service":
		actions = append(actions, ActionDraft{
			ToolName:    "service.restart",
			Description: "Restart the affected service",
			RiskLevel:   "L2",
			Params:      map[string]interface{}{},
		})
	}

	return &DiagnosisResult{
		ProblemType:        plan.ProblemType,
		Confidence:         0.6,
		Findings:           findings,
		RecommendedActions: actions,
		ApprovalRequired:   len(actions) > 0,
		NextStep:           "Review findings and approve recommended actions if appropriate",
	}
}
