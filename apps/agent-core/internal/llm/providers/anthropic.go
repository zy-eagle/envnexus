package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/zy-eagle/envnexus/apps/agent-core/internal/llm/router"
)

type AnthropicProvider struct {
	name    string
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

func NewAnthropicProvider(cfg router.ProviderConfig) *AnthropicProvider {
	name := cfg.Name
	if name == "" {
		name = "anthropic"
	}
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 90 * time.Second
	}
	return &AnthropicProvider{
		name:    name,
		baseURL: baseURL,
		apiKey:  cfg.APIKey,
		model:   cfg.Model,
		client:  &http.Client{Timeout: timeout},
	}
}

func (p *AnthropicProvider) Name() string { return p.name }

func (p *AnthropicProvider) IsAvailable() bool {
	return p.apiKey != "" && p.baseURL != ""
}

type anthropicRequest struct {
	Model       string             `json:"model"`
	MaxTokens   int                `json:"max_tokens"`
	System      string             `json:"system,omitempty"`
	Messages    []anthropicMessage `json:"messages"`
	Temperature float64            `json:"temperature,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Model   string `json:"model"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (p *AnthropicProvider) Complete(ctx context.Context, req *router.CompletionRequest) (*router.CompletionResponse, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 2048
	}
	temp := req.Temperature
	if temp == 0 {
		temp = 0.3
	}

	var systemPrompt string
	var messages []anthropicMessage

	for _, m := range req.Messages {
		if m.Role == "system" {
			systemPrompt = m.Content
			continue
		}
		role := m.Role
		if role != "user" && role != "assistant" {
			role = "user"
		}
		messages = append(messages, anthropicMessage{Role: role, Content: m.Content})
	}

	if len(messages) == 0 {
		return nil, fmt.Errorf("no user/assistant messages provided")
	}

	body := anthropicRequest{
		Model:       model,
		MaxTokens:   maxTokens,
		System:      systemPrompt,
		Messages:    messages,
		Temperature: temp,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/v1/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var aResp anthropicResponse
	if err := json.Unmarshal(respBody, &aResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if aResp.Error != nil {
		return nil, fmt.Errorf("api error [%s]: %s", aResp.Error.Type, aResp.Error.Message)
	}

	var content string
	for _, block := range aResp.Content {
		if block.Type == "text" {
			content += block.Text
		}
	}

	if content == "" {
		return nil, fmt.Errorf("no text content in response")
	}

	return &router.CompletionResponse{
		Content:      content,
		Model:        aResp.Model,
		PromptTokens: aResp.Usage.InputTokens,
		CompTokens:   aResp.Usage.OutputTokens,
	}, nil
}
