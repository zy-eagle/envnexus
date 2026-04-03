package command

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/zy-eagle/envnexus/services/platform-api/internal/domain"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/repository"
)

// nlGenHTTPTimeout is the client timeout for upstream LLM HTTP calls.
// Slow models or congested APIs often exceed ~1m; cap under 5 minutes per product expectation.
const nlGenHTTPTimeout = 4*time.Minute + 45*time.Second

// nlGenMaxTokens leaves room for JSON metadata plus a multi-line shell/PowerShell command.
const nlGenMaxTokens = 512

type NLGenerator struct {
	modelRepo  repository.ModelProfileRepository
	httpClient *http.Client
}

func NewNLGenerator(modelRepo repository.ModelProfileRepository) *NLGenerator {
	return &NLGenerator{
		modelRepo:  modelRepo,
		httpClient: &http.Client{Timeout: nlGenHTTPTimeout},
	}
}

type GenerateCommandResult struct {
	Command   string `json:"command"`
	RiskLevel string `json:"risk_level,omitempty"`
	Title     string `json:"title,omitempty"`
}

const systemPrompt = `You are an operations assistant. Convert user requests into a single executable shell command. Respond with ONLY a JSON object:
{"command":"<shell command>","risk_level":"<L1|L2|L3>","title":"<short title>"}

Rules:
- Prefer cross-platform commands when possible.
- For multiple steps, prefer newlines (\n) or PowerShell ";" separators. Avoid using "&&" because it is not supported by Windows PowerShell 5.x.
- No markdown, no explanation.`

func (g *NLGenerator) Generate(ctx context.Context, tenantID, prompt string) (*GenerateCommandResult, error) {
	start := time.Now()
	model, err := g.pickModel(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("no available model: %w", err)
	}
	slog.Info("[nl-gen] picked model", "model", model.ModelName, "elapsed_ms", time.Since(start).Milliseconds())

	llmStart := time.Now()
	respText, err := g.callLLM(ctx, model, prompt)
	if err != nil {
		var appErr *domain.AppError
		if errors.As(err, &appErr) {
			return nil, err
		}
		return nil, fmt.Errorf("LLM call failed (took %dms): %w", time.Since(llmStart).Milliseconds(), err)
	}
	slog.Info("[nl-gen] LLM responded", "elapsed_ms", time.Since(llmStart).Milliseconds(), "total_ms", time.Since(start).Milliseconds())

	var result GenerateCommandResult
	cleaned := normalizeLLMTextForJSON(respText)
	if cleaned == "" {
		cleaned = strings.TrimSpace(respText)
	}

	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		if alt := firstJSONObject(respText); alt != "" && alt != cleaned {
			cleaned = normalizeLLMTextForJSON(alt)
			err = json.Unmarshal([]byte(cleaned), &result)
		}
		if err != nil {
			slog.Warn("[nl-gen] Failed to parse LLM JSON, using raw text as command", "raw", respText)
			result.Command = strings.TrimSpace(respText)
			result.RiskLevel = EvaluateRisk("shell", result.Command)
		}
	}
	// Some OpenAI-compatible providers may return a different JSON shape (e.g. {"cmd": "..."}).
	// If we parsed JSON but the command is still empty, try a generic map-based extraction.
	if strings.TrimSpace(result.Command) == "" && strings.HasPrefix(cleaned, "{") {
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(cleaned), &m); err == nil {
			for _, k := range []string{"command", "cmd", "shell", "bash", "powershell", "executable", "script", "line", "powershell_command"} {
				if v, ok := m[k]; ok {
					if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
						result.Command = strings.TrimSpace(s)
						break
					}
				}
			}
		}
	}
	// If the provider returned an empty content despite a 200 OK, surface a clear error.
	if strings.TrimSpace(result.Command) == "" {
		slog.Warn("[nl-gen] Empty command from LLM", "raw", respText)
		msg := "模型未返回可解析的命令内容"
		if preview := truncateForDisplay(respText, 800); preview != "" {
			msg += ": " + preview
		}
		return nil, domain.NewAppError("llm_empty_command", msg, http.StatusUnprocessableEntity)
	}

	if result.RiskLevel == "" {
		result.RiskLevel = EvaluateRisk("shell", result.Command)
	}

	slog.Info("[nl-gen] result", "command_len", len(result.Command), "risk", result.RiskLevel, "title", result.Title, "total_ms", time.Since(start).Milliseconds())
	return &result, nil
}

func (g *NLGenerator) pickModel(ctx context.Context, tenantID string) (*domain.ModelProfile, error) {
	profiles, err := g.modelRepo.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	for _, p := range profiles {
		if p.Status == "active" && p.BaseURL != "" && p.APIKey != "" {
			return p, nil
		}
	}
	return nil, fmt.Errorf("no active model profile with API key configured for tenant %s", tenantID)
}

// normalizeLLMTextForJSON strips common markdown fences and whitespace around a JSON payload.
func normalizeLLMTextForJSON(s string) string {
	cleaned := strings.TrimSpace(s)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	return strings.TrimSpace(cleaned)
}

// firstJSONObject returns the substring of the first top-level JSON object in s, if any.
func firstJSONObject(s string) string {
	i := strings.Index(s, "{")
	if i < 0 {
		return ""
	}
	depth := 0
	inString := false
	escape := false
	for j := i; j < len(s); j++ {
		ch := s[j]
		if inString {
			if escape {
				escape = false
				continue
			}
			if ch == '\\' {
				escape = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}
		switch ch {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[i : j+1]
			}
		}
	}
	return ""
}

func truncateForDisplay(s string, maxRunes int) string {
	s = strings.TrimSpace(s)
	if maxRunes <= 0 {
		return s
	}
	r := []rune(s)
	if len(r) <= maxRunes {
		return s
	}
	return string(r[:maxRunes]) + "…"
}

// extractOpenAICompatError reads provider error shapes such as {"error":{"message":"...","type":"..."}}.
func extractOpenAICompatError(raw []byte) (string, bool) {
	var envelope struct {
		Error *struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		} `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return "", false
	}
	if envelope.Error != nil && strings.TrimSpace(envelope.Error.Message) != "" {
		msg := strings.TrimSpace(envelope.Error.Message)
		if t := strings.TrimSpace(envelope.Error.Type); t != "" {
			msg = t + ": " + msg
		}
		return msg, true
	}
	if strings.TrimSpace(envelope.Message) != "" {
		return strings.TrimSpace(envelope.Message), true
	}
	return "", false
}

func decodeAssistantContent(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	// Array of heterogeneous parts (OpenAI / Anthropic / DeepSeek / gateways).
	var rawParts []json.RawMessage
	if err := json.Unmarshal(raw, &rawParts); err == nil && len(rawParts) > 0 {
		var b strings.Builder
		for _, p := range rawParts {
			// Avoid infinite recursion on empty
			if len(p) > 0 && string(p) != "null" {
				b.WriteString(decodeAssistantContent(p))
			}
		}
		return b.String()
	}
	var parts []map[string]interface{}
	if err := json.Unmarshal(raw, &parts); err != nil {
		return ""
	}
	var b strings.Builder
	for _, p := range parts {
		typ, _ := p["type"].(string)
		// text (OpenAI, many gateways), output_text (responses API), reasoning (optional strip handled by caller priority)
		switch typ {
		case "text", "output_text", "input_text", "":
			if txt, ok := p["text"].(string); ok {
				b.WriteString(txt)
			} else if c, ok := p["content"].(string); ok {
				b.WriteString(c)
			}
		case "reasoning":
			// DeepSeek / o1-style block: prefer explicit summary field if present
			if txt, ok := p["text"].(string); ok {
				b.WriteString(txt)
			} else if txt, ok := p["summary"].(string); ok {
				b.WriteString(txt)
			}
		default:
			// Unknown block: still try common text fields (provider-specific types).
			if txt, ok := p["text"].(string); ok {
				b.WriteString(txt)
			} else if c, ok := p["content"].(string); ok {
				b.WriteString(c)
			}
		}
	}
	return b.String()
}

// stringifyContentField turns provider "content" / "reasoning_content" values into plain text.
func stringifyContentField(v interface{}) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case []interface{}:
		var b strings.Builder
		for _, item := range x {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			part, _ := json.Marshal(m)
			b.WriteString(decodeAssistantContent(part))
		}
		return b.String()
	case map[string]interface{}:
		// Rare: single object instead of array
		part, _ := json.Marshal(x)
		return decodeAssistantContent(part)
	default:
		return ""
	}
}

func textFromChoiceMap(choice map[string]interface{}) string {
	if choice == nil {
		return ""
	}
	if msg, ok := choice["message"].(map[string]interface{}); ok {
		if s := stringifyContentField(msg["content"]); strings.TrimSpace(s) != "" {
			return s
		}
		// DeepSeek-V3/R1 reasoner: chain-of-thought; final answer should be in content, but some gateways leave content empty.
		if s := stringifyContentField(msg["reasoning_content"]); strings.TrimSpace(s) != "" {
			return s
		}
	}
	if d, ok := choice["delta"].(map[string]interface{}); ok {
		if s := stringifyContentField(d["content"]); strings.TrimSpace(s) != "" {
			return s
		}
		if s := stringifyContentField(d["reasoning_content"]); strings.TrimSpace(s) != "" {
			return s
		}
	}
	if t, ok := choice["text"].(string); ok {
		return t
	}
	return ""
}

// extractAssistantTextFromRawResponse walks unknown OpenAI-compatible JSON (DeepSeek, OpenRouter, proxies) for assistant text.
func extractAssistantTextFromRawResponse(raw []byte) string {
	var root map[string]interface{}
	if err := json.Unmarshal(raw, &root); err != nil {
		return ""
	}
	choices, ok := root["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return ""
	}
	for _, ch := range choices {
		cm, ok := ch.(map[string]interface{})
		if !ok {
			continue
		}
		if s := textFromChoiceMap(cm); strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

func (g *NLGenerator) callLLM(ctx context.Context, model *domain.ModelProfile, userMessage string) (string, error) {
	baseURL := strings.TrimSuffix(model.BaseURL, "/")

	body := map[string]interface{}{
		"model": model.ModelName,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userMessage},
		},
		"temperature": 0.1,
		"max_tokens":  nlGenMaxTokens,
	}

	payload, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+model.APIKey)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	raw, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return "", readErr
	}

	if resp.StatusCode >= 400 {
		if msg, ok := extractOpenAICompatError(raw); ok {
			return "", domain.NewAppError("llm_provider_error",
				fmt.Sprintf("%s (HTTP %d)", msg, resp.StatusCode), http.StatusBadGateway)
		}
		return "", domain.NewAppError("llm_provider_error",
			fmt.Sprintf("LLM API HTTP %d: %s", resp.StatusCode, truncateForDisplay(string(raw), 1200)),
			http.StatusBadGateway)
	}

	var chatResp struct {
		Choices []struct {
			FinishReason string `json:"finish_reason"`
			// OpenAI Chat Completions (+ DeepSeek reasoning fields on message)
			Message *struct {
				Content          json.RawMessage `json:"content"`
				ReasoningContent json.RawMessage `json:"reasoning_content"`
			} `json:"message,omitempty"`
			// Some providers may respond with legacy "text" on each choice.
			Text *string `json:"text,omitempty"`
			// Streaming-like delta shape (occasionally returned even without stream=true).
			Delta *struct {
				Content          json.RawMessage `json:"content"`
				ReasoningContent json.RawMessage `json:"reasoning_content"`
			} `json:"delta,omitempty"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &chatResp); err != nil {
		if fallback := strings.TrimSpace(extractAssistantTextFromRawResponse(raw)); fallback != "" {
			slog.Info("[nl-gen] used generic JSON fallback for chat response")
			return fallback, nil
		}
		return "", fmt.Errorf("parse LLM response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		if msg, ok := extractOpenAICompatError(raw); ok {
			return "", domain.NewAppError("llm_provider_error", msg, http.StatusBadGateway)
		}
		if fallback := strings.TrimSpace(extractAssistantTextFromRawResponse(raw)); fallback != "" {
			return fallback, nil
		}
		return "", domain.NewAppError("llm_provider_error",
			"模型未返回任何候选回复 (choices 为空)", http.StatusBadGateway)
	}

	for i := range chatResp.Choices {
		c0 := chatResp.Choices[i]
		if c0.Message != nil {
			if len(c0.Message.Content) > 0 {
				if t := decodeAssistantContent(c0.Message.Content); strings.TrimSpace(t) != "" {
					return t, nil
				}
			}
			if len(c0.Message.ReasoningContent) > 0 {
				if t := decodeAssistantContent(c0.Message.ReasoningContent); strings.TrimSpace(t) != "" {
					return t, nil
				}
			}
		}
		if c0.Delta != nil {
			if len(c0.Delta.Content) > 0 {
				if t := decodeAssistantContent(c0.Delta.Content); strings.TrimSpace(t) != "" {
					return t, nil
				}
			}
			if len(c0.Delta.ReasoningContent) > 0 {
				if t := decodeAssistantContent(c0.Delta.ReasoningContent); strings.TrimSpace(t) != "" {
					return t, nil
				}
			}
		}
		if c0.Text != nil && strings.TrimSpace(*c0.Text) != "" {
			return *c0.Text, nil
		}
	}

	// Structured decode missed text (unusual types); try map walk last.
	if fallback := strings.TrimSpace(extractAssistantTextFromRawResponse(raw)); fallback != "" {
		slog.Info("[nl-gen] used generic JSON fallback after empty structured fields", "choice_count", len(chatResp.Choices))
		return fallback, nil
	}

	c0 := chatResp.Choices[0]
	if msg, ok := extractOpenAICompatError(raw); ok {
		slog.Warn("[nl-gen] provider error in body without assistant text", "raw_response_len", len(raw))
		return "", domain.NewAppError("llm_provider_error", msg, http.StatusBadGateway)
	}
	slog.Warn("[nl-gen] LLM choice had no usable text", "raw_response_len", len(raw))
	fr := strings.TrimSpace(c0.FinishReason)
	if fr != "" {
		return "", domain.NewAppError("llm_provider_error",
			"模型未返回可解析文本 (finish_reason="+fr+")", http.StatusBadGateway)
	}
	return "", domain.NewAppError("llm_provider_error", "模型未返回可解析文本", http.StatusBadGateway)
}
