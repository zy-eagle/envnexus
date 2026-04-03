package command

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/zy-eagle/envnexus/services/platform-api/internal/domain"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/repository"
)

type NLGenerator struct {
	modelRepo  repository.ModelProfileRepository
	httpClient *http.Client
}

func NewNLGenerator(modelRepo repository.ModelProfileRepository) *NLGenerator {
	return &NLGenerator{
		modelRepo:  modelRepo,
		httpClient: &http.Client{Timeout: 55 * time.Second},
	}
}

type GenerateCommandResult struct {
	Command   string `json:"command"`
	RiskLevel string `json:"risk_level,omitempty"`
	Title     string `json:"title,omitempty"`
}

const systemPrompt = `You are an operations assistant. Convert user requests into shell commands. Respond with ONLY a JSON object:
{"command":"<shell command>","risk_level":"<L1|L2|L3>","title":"<short title>"}
L1=read-only, L2=service ops, L3=destructive. For multiple steps, chain with && or use newlines (\n). No markdown, no explanation.`

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
		return nil, fmt.Errorf("LLM call failed (took %dms): %w", time.Since(llmStart).Milliseconds(), err)
	}
	slog.Info("[nl-gen] LLM responded", "elapsed_ms", time.Since(llmStart).Milliseconds(), "total_ms", time.Since(start).Milliseconds())

	var result GenerateCommandResult
	cleaned := strings.TrimSpace(respText)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		slog.Warn("[nl-gen] Failed to parse LLM JSON, using raw text as command", "raw", respText)
		result.Command = strings.TrimSpace(respText)
		result.RiskLevel = EvaluateRisk("shell", result.Command)
	}
	// Some OpenAI-compatible providers may return a different JSON shape (e.g. {"cmd": "..."}).
	// If we parsed JSON but the command is still empty, try a generic map-based extraction.
	if strings.TrimSpace(result.Command) == "" && strings.HasPrefix(cleaned, "{") {
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(cleaned), &m); err == nil {
			for _, k := range []string{"command", "cmd", "shell", "bash", "powershell"} {
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
		return nil, fmt.Errorf("LLM returned empty command")
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

func (g *NLGenerator) callLLM(ctx context.Context, model *domain.ModelProfile, userMessage string) (string, error) {
	baseURL := strings.TrimSuffix(model.BaseURL, "/")

	body := map[string]interface{}{
		"model": model.ModelName,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userMessage},
		},
		"temperature": 0.1,
		"max_tokens":  256,
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

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("LLM API returned %d: %s", resp.StatusCode, string(raw))
	}

	var chatResp struct {
		Choices []struct {
			// OpenAI Chat Completions
			Message *struct {
				Content *string `json:"content"`
			} `json:"message,omitempty"`
			// Some providers may respond with legacy "text" on each choice.
			Text *string `json:"text,omitempty"`
			// Streaming-like delta shape (occasionally returned even without stream=true).
			Delta *struct {
				Content *string `json:"content"`
			} `json:"delta,omitempty"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &chatResp); err != nil {
		return "", fmt.Errorf("parse LLM response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("LLM returned no choices")
	}

	c0 := chatResp.Choices[0]
	if c0.Message != nil && c0.Message.Content != nil {
		return *c0.Message.Content, nil
	}
	if c0.Delta != nil && c0.Delta.Content != nil {
		return *c0.Delta.Content, nil
	}
	if c0.Text != nil {
		return *c0.Text, nil
	}
	return "", fmt.Errorf("LLM returned empty content")
}
