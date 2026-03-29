package diagnosis

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/zy-eagle/envnexus/apps/agent-core/internal/llm/router"
	"github.com/zy-eagle/envnexus/apps/agent-core/internal/tools"
)

type DiagnosisPlan struct {
	ProblemType string                            `json:"problem_type"`
	Scope       string                            `json:"scope"`
	RiskBias    string                            `json:"risk_bias"`
	ToolNames   []string                          `json:"tool_names"`
	ToolParams  map[string]map[string]interface{} `json:"tool_params,omitempty"`
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

type ProgressFn func(step string, detail string)

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
	return e.PlanWithProgress(ctx, sessionID, input, nil)
}

func (e *Engine) PlanWithProgress(ctx context.Context, sessionID, input string, onProgress ProgressFn) (*DiagnosisPlan, error) {
	slog.Info("[diagnosis] Planning", "session_id", sessionID, "input", input)

	notify(onProgress, "intent_parse", "Analyzing problem type...")
	plan, err := e.stepIntentParse(ctx, input)
	if err != nil {
		slog.Warn("[diagnosis] IntentParse failed, using heuristic", "error", err)
		notify(onProgress, "intent_parse_fallback", "Using local heuristic analysis")
		plan = e.heuristicPlan(input)
	}

	plan.ToolNames = e.stepEvidencePlan(plan)
	notify(onProgress, "plan_ready", fmt.Sprintf("Problem type: %s, tools: %d", plan.ProblemType, len(plan.ToolNames)))
	slog.Info("[diagnosis] Plan", "problem_type", plan.ProblemType, "tools", plan.ToolNames)
	return plan, nil
}

func (e *Engine) Execute(ctx context.Context, sessionID string, plan *DiagnosisPlan) (*DiagnosisResult, error) {
	return e.ExecuteWithProgress(ctx, sessionID, plan, nil)
}

func (e *Engine) ExecuteWithProgress(ctx context.Context, sessionID string, plan *DiagnosisPlan, onProgress ProgressFn) (*DiagnosisResult, error) {
	start := time.Now()
	slog.Info("[diagnosis] Executing plan", "session_id", sessionID, "problem_type", plan.ProblemType)

	notify(onProgress, "evidence_collect", fmt.Sprintf("Collecting evidence from %d tools...", len(plan.ToolNames)))
	evidence := e.stepEvidenceCollect(ctx, plan)
	notify(onProgress, "evidence_done", fmt.Sprintf("Collected %d evidence items", len(evidence)))

	notify(onProgress, "reasoning", "Generating diagnosis...")
	reasoning, err := e.stepReasoning(ctx, plan, evidence)
	if err != nil {
		slog.Warn("[diagnosis] Reasoning via LLM failed, using local analysis", "error", err)
		notify(onProgress, "reasoning_fallback", "Using local analysis engine")
		reasoning = e.localReasoning(plan, evidence)
	}

	reasoning.Evidence = evidence
	reasoning.DurationMs = time.Since(start).Milliseconds()

	if len(reasoning.RecommendedActions) > 0 {
		e.stepActionDraft(reasoning)
	}

	notify(onProgress, "complete", fmt.Sprintf("Done in %dms", reasoning.DurationMs))
	slog.Info("[diagnosis] Complete",
		"problem_type", reasoning.ProblemType,
		"confidence", reasoning.Confidence,
		"findings", len(reasoning.Findings),
		"actions", len(reasoning.RecommendedActions),
		"duration_ms", reasoning.DurationMs,
	)

	return reasoning, nil
}

func (e *Engine) RunDiagnosis(ctx context.Context, sessionID, intent string) (*DiagnosisResult, error) {
	return e.RunDiagnosisWithProgress(ctx, sessionID, intent, nil)
}

func (e *Engine) RunDiagnosisWithProgress(ctx context.Context, sessionID, intent string, onProgress ProgressFn) (*DiagnosisResult, error) {
	plan, err := e.PlanWithProgress(ctx, sessionID, intent, onProgress)
	if err != nil {
		return nil, fmt.Errorf("plan: %w", err)
	}
	return e.ExecuteWithProgress(ctx, sessionID, plan, onProgress)
}

func notify(fn ProgressFn, step, detail string) {
	if fn != nil {
		fn(step, detail)
	}
}

// extractJSON finds the first valid JSON object in a string that may contain
// mixed natural language and JSON. This handles DeepSeek R1 models that return
// reasoning text with embedded JSON.
func extractJSON(raw string) string {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(s)

	if len(s) > 0 && s[0] == '{' {
		return s
	}

	start := strings.Index(s, "{")
	if start == -1 {
		return s
	}

	depth := 0
	for i := start; i < len(s); i++ {
		switch s[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}

	return s[start:]
}

// Step 1: IntentParse — use LLM to classify the problem
func (e *Engine) stepIntentParse(ctx context.Context, input string) (*DiagnosisPlan, error) {
	if e.llmRouter == nil {
		return nil, fmt.Errorf("no LLM router configured")
	}

	prompt := fmt.Sprintf(`You are an IT diagnostic assistant. Analyze the user's problem description and return a JSON object with:
- "problem_type": one of "network", "dns", "service", "performance", "disk", "auth", "install", "general"
- "scope": one of "local", "network", "system"
- "risk_bias": one of "conservative", "moderate", "aggressive"

User's problem: %s

You MUST respond with ONLY a JSON object. No explanation, no markdown, no thinking process. Just the raw JSON object starting with { and ending with }.`, input)

	resp, err := e.llmRouter.Complete(ctx, &router.CompletionRequest{
		Messages: []router.Message{
			{Role: "system", Content: "You are a structured diagnostic classifier. You MUST output ONLY a valid JSON object. No explanation, no markdown fences, no thinking. Just raw JSON."},
			{Role: "user", Content: prompt},
		},
		MaxTokens:   256,
		Temperature: 0.1,
	})
	if err != nil {
		return nil, err
	}

	var plan DiagnosisPlan
	content := extractJSON(resp.Content)
	slog.Debug("[diagnosis] IntentParse raw response", "content_len", len(resp.Content), "extracted_json", content[:min(len(content), 200)])

	if err := json.Unmarshal([]byte(content), &plan); err != nil {
		return nil, fmt.Errorf("parse LLM response: %w (content: %s)", err, content[:min(len(content), 300)])
	}
	return &plan, nil
}

// Step 2: EvidencePlan — select read-only tools based on problem type
func (e *Engine) stepEvidencePlan(plan *DiagnosisPlan) []string {
	toolMapping := map[string][]string{
		"network":     {"read_network_config", "read_proxy_config", "read_system_info", "ping_host", "dns_lookup", "read_route_table", "read_env_vars"},
		"dns":         {"read_network_config", "read_system_info", "dns_lookup", "read_env_vars"},
		"service":     {"read_system_info", "read_process_list", "read_env_vars", "http_check", "read_event_log", "check_runtime_deps"},
		"performance": {"read_system_info", "read_process_list", "read_disk_usage"},
		"disk":        {"read_system_info", "read_disk_usage", "read_dir_list"},
		"auth":        {"read_system_info", "read_env_vars", "check_tls_cert"},
		"install":     {"read_system_info", "read_disk_usage", "read_installed_apps", "check_runtime_deps", "read_event_log", "read_env_vars", "read_file_info"},
		"general":     {"read_system_info", "read_network_config", "read_proxy_config", "read_env_vars"},
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
func (e *Engine) stepEvidenceCollect(ctx context.Context, plan *DiagnosisPlan) map[string]interface{} {
	evidence := make(map[string]interface{})
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, name := range plan.ToolNames {
		t, ok := e.registry.Get(name)
		if !ok {
			continue
		}
		var params map[string]interface{}
		if plan.ToolParams != nil {
			params = plan.ToolParams[name]
		}
		wg.Add(1)
		go func(tool tools.Tool, toolName string, toolParams map[string]interface{}) {
			defer wg.Done()
			result, err := tool.Execute(ctx, toolParams)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				evidence[toolName] = map[string]interface{}{"error": err.Error()}
				slog.Warn("[diagnosis] Tool failed during evidence collection", "tool", toolName, "error", err)
			} else {
				evidence[toolName] = result.Output
				slog.Info("[diagnosis] Tool collected", "tool", toolName, "summary", result.Summary)
			}
		}(t, name, params)
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

	var toolList strings.Builder
	for _, t := range e.registry.List() {
		toolList.WriteString(fmt.Sprintf("- %s: %s (read_only=%v, risk=%s)\n", t.Name(), t.Description(), t.IsReadOnly(), t.RiskLevel()))
	}

	prompt := fmt.Sprintf(`You are an IT diagnostic engine. Given the problem type and collected evidence, produce a diagnosis.

Problem type: %s
Scope: %s

Evidence collected:
%s

Available tools (ONLY use these tool_name values in recommended_actions):
%s

Return a JSON object with:
- "problem_type": string
- "confidence": float between 0.0 and 1.0
- "findings": array of {"source": string, "summary": string, "level": "info"|"warning"|"error"}
- "recommended_actions": array of {"tool_name": string, "description": string, "risk_level": "L0"|"L1"|"L2"|"L3", "params": {}}. IMPORTANT: tool_name MUST be one of the available tools listed above. Do NOT invent tool names.
- "approval_required": boolean (true if any action is L1 or above)
- "next_step": string describing what to do next

You MUST respond with ONLY a JSON object. No explanation, no markdown, no thinking process. Just the raw JSON object starting with { and ending with }.`, plan.ProblemType, plan.Scope, string(evidenceJSON), toolList.String())

	resp, err := e.llmRouter.Complete(ctx, &router.CompletionRequest{
		Messages: []router.Message{
			{Role: "system", Content: "You are a structured IT diagnostic engine. You MUST output ONLY a valid JSON object. No explanation, no markdown fences, no thinking. Just raw JSON."},
			{Role: "user", Content: prompt},
		},
		MaxTokens:   2048,
		Temperature: 0.2,
	})
	if err != nil {
		return nil, err
	}

	var result DiagnosisResult
	content := extractJSON(resp.Content)
	slog.Debug("[diagnosis] Reasoning raw response", "content_len", len(resp.Content), "extracted_json", content[:min(len(content), 200)])

	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("parse reasoning response: %w (content: %s)", err, content[:min(len(content), 300)])
	}
	return &result, nil
}

// Step 5: ActionDraft — validate and enrich recommended actions
func (e *Engine) stepActionDraft(result *DiagnosisResult) {
	var validated []ActionDraft
	for _, action := range result.RecommendedActions {
		t, ok := e.registry.Get(action.ToolName)
		if !ok {
			slog.Warn("[diagnosis] ActionDraft: tool not found, skipping", "tool", action.ToolName)
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

var ipOrHostRe = regexp.MustCompile(`(?:(?:\d{1,3}\.){3}\d{1,3})|(?:[a-zA-Z0-9](?:[a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z]{2,})+)`)

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
	case strings.Contains(lower, "proxy") || strings.Contains(lower, "代理") ||
		strings.Contains(lower, "network") || strings.Contains(lower, "网络") ||
		strings.Contains(lower, "connect") || strings.Contains(lower, "连接") ||
		strings.Contains(lower, "ping") || strings.Contains(lower, "延迟") ||
		strings.Contains(lower, "timeout") || strings.Contains(lower, "超时") ||
		strings.Contains(lower, "vpn") || strings.Contains(lower, "防火墙") || strings.Contains(lower, "firewall"):
		plan.ProblemType = "network"
		plan.Scope = "network"
	case strings.Contains(lower, "service") || strings.Contains(lower, "服务") || strings.Contains(lower, "restart") || strings.Contains(lower, "重启"):
		plan.ProblemType = "service"
	case strings.Contains(lower, "slow") || strings.Contains(lower, "慢") || strings.Contains(lower, "performance") ||
		strings.Contains(lower, "cpu") || strings.Contains(lower, "内存") || strings.Contains(lower, "memory") || strings.Contains(lower, "卡"):
		plan.ProblemType = "performance"
	case strings.Contains(lower, "disk") || strings.Contains(lower, "磁盘") || strings.Contains(lower, "space") ||
		strings.Contains(lower, "存储") || strings.Contains(lower, "容量"):
		plan.ProblemType = "disk"
	case strings.Contains(lower, "auth") || strings.Contains(lower, "认证") || strings.Contains(lower, "登录") ||
		strings.Contains(lower, "password") || strings.Contains(lower, "密码") || strings.Contains(lower, "权限"):
		plan.ProblemType = "auth"
	case strings.Contains(lower, "install") || strings.Contains(lower, "安装") || strings.Contains(lower, "setup") ||
		strings.Contains(lower, "卸载") || strings.Contains(lower, "uninstall") || strings.Contains(lower, "依赖") ||
		strings.Contains(lower, "dependency") || strings.Contains(lower, "runtime") || strings.Contains(lower, "运行时") ||
		strings.Contains(lower, "dll") || strings.Contains(lower, "缺少") || strings.Contains(lower, "missing") ||
		strings.Contains(lower, "版本") || strings.Contains(lower, "version") || strings.Contains(lower, "兼容") ||
		strings.Contains(lower, "compatible"):
		plan.ProblemType = "install"
		plan.Scope = "local"
	}

	if host := ipOrHostRe.FindString(input); host != "" {
		plan.ToolParams = map[string]map[string]interface{}{
			"ping_host": {"host": host},
		}
	}

	return plan
}

func (e *Engine) localReasoning(plan *DiagnosisPlan, evidence map[string]interface{}) *DiagnosisResult {
	findings := make([]Finding, 0)

	for source, data := range evidence {
		if dataMap, ok := data.(map[string]interface{}); ok {
			if errMsg, exists := dataMap["error"]; exists {
				findings = append(findings, Finding{Source: source, Summary: fmt.Sprintf("Collection failed: %v", errMsg), Level: "error"})
				continue
			}
		}

		switch source {
		case "read_proxy_config":
			if dataMap, ok := data.(map[string]interface{}); ok {
				findings = append(findings, e.analyzeProxyEvidence(dataMap)...)
			}
		case "read_network_config":
			findings = append(findings, e.analyzeNetworkEvidence(data)...)
		case "read_system_info":
			if dataMap, ok := data.(map[string]interface{}); ok {
				findings = append(findings, e.analyzeSystemEvidence(dataMap)...)
			}
		case "ping_host":
			if dataMap, ok := data.(map[string]interface{}); ok {
				findings = append(findings, e.analyzePingEvidence(dataMap)...)
			}
		default:
			raw, _ := json.MarshalIndent(data, "", "  ")
			findings = append(findings, Finding{Source: source, Summary: string(raw), Level: "info"})
		}
	}

	if len(findings) == 0 {
		findings = append(findings, Finding{Source: "diagnosis", Summary: "No evidence could be collected", Level: "warning"})
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

func (e *Engine) analyzeProxyEvidence(data map[string]interface{}) []Finding {
	var findings []Finding
	hasProxy, _ := data["has_proxy"].(bool)

	if hasProxy {
		findings = append(findings, Finding{Source: "read_proxy_config", Summary: "Proxy is configured on this system", Level: "info"})
		switch envProxies := data["env_proxies"].(type) {
		case map[string]string:
			for k, v := range envProxies {
				findings = append(findings, Finding{Source: "read_proxy_config", Summary: fmt.Sprintf("Environment variable %s = %s", k, v), Level: "info"})
			}
		case map[string]interface{}:
			for k, v := range envProxies {
				findings = append(findings, Finding{Source: "read_proxy_config", Summary: fmt.Sprintf("Environment variable %s = %v", k, v), Level: "info"})
			}
		}
		if goProxy, ok := data["go_http_proxy"].(string); ok && goProxy != "" {
			findings = append(findings, Finding{Source: "read_proxy_config", Summary: fmt.Sprintf("Go HTTP transport proxy: %s", goProxy), Level: "info"})
		}
		if sysProxy, ok := data["system_proxy"].(string); ok && sysProxy != "" {
			findings = append(findings, Finding{Source: "read_proxy_config", Summary: fmt.Sprintf("System proxy: %s", sysProxy), Level: "info"})
		}
	} else {
		findings = append(findings, Finding{Source: "read_proxy_config", Summary: "No proxy detected — no HTTP_PROXY/HTTPS_PROXY environment variables set, no system proxy configured", Level: "info"})
	}
	return findings
}

func (e *Engine) analyzeNetworkEvidence(data interface{}) []Finding {
	var findings []Finding

	var interfaces []interface{}
	switch v := data.(type) {
	case []interface{}:
		interfaces = v
	case []map[string]interface{}:
		for _, m := range v {
			interfaces = append(interfaces, m)
		}
	default:
		raw, _ := json.Marshal(data)
		findings = append(findings, Finding{Source: "read_network_config", Summary: string(raw), Level: "info"})
		return findings
	}

	activeCount := 0
	for _, iface := range interfaces {
		ifMap, ok := iface.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := ifMap["name"].(string)
		flags, _ := ifMap["flags"].(string)

		var ipStrs []string
		switch v := ifMap["ip_addresses"].(type) {
		case []string:
			ipStrs = v
		case []interface{}:
			for _, ip := range v {
				ipStrs = append(ipStrs, fmt.Sprintf("%v", ip))
			}
		}

		if len(ipStrs) == 0 {
			continue
		}

		if strings.Contains(flags, "up") {
			activeCount++
			findings = append(findings, Finding{
				Source:  "read_network_config",
				Summary: fmt.Sprintf("Interface %s (UP): %s", name, strings.Join(ipStrs, ", ")),
				Level:   "info",
			})
		}
	}

	if activeCount == 0 {
		findings = append(findings, Finding{Source: "read_network_config", Summary: "No active network interfaces found", Level: "warning"})
	} else {
		findings = append(findings, Finding{
			Source:  "read_network_config",
			Summary: fmt.Sprintf("Found %d active network interface(s)", activeCount),
			Level:   "info",
		})
	}
	return findings
}

func (e *Engine) analyzeSystemEvidence(data map[string]interface{}) []Finding {
	osName, _ := data["os"].(string)
	arch, _ := data["architecture"].(string)
	hostname, _ := data["hostname"].(string)
	numCPU := 0
	switch n := data["num_cpu"].(type) {
	case int:
		numCPU = n
	case float64:
		numCPU = int(n)
	}

	return []Finding{
		{
			Source:  "read_system_info",
			Summary: fmt.Sprintf("System: %s/%s, Hostname: %s, CPUs: %d", osName, arch, hostname, numCPU),
			Level:   "info",
		},
	}
}

func (e *Engine) analyzePingEvidence(data map[string]interface{}) []Finding {
	host, _ := data["host"].(string)
	port, _ := data["port"].(string)
	reachable, _ := data["reachable"].(bool)

	var latency int64
	switch v := data["latency_ms"].(type) {
	case float64:
		latency = int64(v)
	case int64:
		latency = v
	}

	if reachable {
		return []Finding{{
			Source:  "ping_host",
			Summary: fmt.Sprintf("Host %s:%s is reachable (latency: %dms)", host, port, latency),
			Level:   "info",
		}}
	}

	errMsg, _ := data["error"].(string)
	return []Finding{{
		Source:  "ping_host",
		Summary: fmt.Sprintf("Host %s:%s is NOT reachable (%s)", host, port, errMsg),
		Level:   "warning",
	}}
}

func (e *Engine) analyzeSliceEvidence(source string, data []interface{}) []Finding {
	raw, _ := json.MarshalIndent(data, "", "  ")
	return []Finding{{Source: source, Summary: string(raw), Level: "info"}}
}
