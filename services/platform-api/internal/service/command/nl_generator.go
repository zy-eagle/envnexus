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
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

type GenerateCommandResult struct {
	Command   string `json:"command"`
	RiskLevel string `json:"risk_level,omitempty"`
	Title     string `json:"title,omitempty"`
}

const systemPrompt = `You are a Linux/Windows operations assistant. The user will describe what they want to do in natural language. You must respond with ONLY a valid JSON object (no markdown, no explanation), containing:
- "command": the exact shell command(s) to execute
- "risk_level": one of "L1" (read-only/safe), "L2" (service restart/moderate), "L3" (destructive/dangerous)
- "title": a short title (under 60 chars) describing the operation

Examples:
User: "Restart the nginx service"
{"command":"systemctl restart nginx","risk_level":"L2","title":"Restart nginx service"}

User: "Check disk usage"
{"command":"df -h","risk_level":"L1","title":"Check disk usage"}

User: "Delete all log files older than 30 days"
{"command":"find /var/log -name '*.log' -mtime +30 -delete","risk_level":"L3","title":"Clean old log files"}

Respond with ONLY the JSON object.`

func (g *NLGenerator) Generate(ctx context.Context, tenantID, prompt string) (*GenerateCommandResult, error) {
	model, err := g.pickModel(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("no available model: %w", err)
	}

	respText, err := g.callLLM(ctx, model, prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

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

	if result.RiskLevel == "" {
		result.RiskLevel = EvaluateRisk("shell", result.Command)
	}

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
		"max_tokens":  512,
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
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &chatResp); err != nil {
		return "", fmt.Errorf("parse LLM response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("LLM returned no choices")
	}

	return chatResp.Choices[0].Message.Content, nil
}
