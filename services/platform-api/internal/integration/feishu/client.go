package feishu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

const (
	baseURL            = "https://open.feishu.cn/open-apis"
	tokenEndpoint      = "/auth/v3/tenant_access_token/internal"
	sendMessageURL     = "/im/v1/messages"
	interactiveCardURL = "/im/v1/messages"
)

// Client is a Feishu (Lark) Open Platform API client.
type Client struct {
	appID     string
	appSecret string
	httpClient *http.Client

	mu          sync.RWMutex
	accessToken string
	tokenExpiry time.Time
}

// NewClient creates a new Feishu API client.
func NewClient(appID, appSecret string) *Client {
	return &Client{
		appID:     appID,
		appSecret: appSecret,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Enabled returns true if the Feishu integration is configured.
func (c *Client) Enabled() bool {
	return c.appID != "" && c.appSecret != ""
}

func (c *Client) getAccessToken(ctx context.Context) (string, error) {
	c.mu.RLock()
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		token := c.accessToken
		c.mu.RUnlock()
		return token, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		return c.accessToken, nil
	}

	body, _ := json.Marshal(map[string]string{
		"app_id":     c.appID,
		"app_secret": c.appSecret,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+tokenEndpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch tenant access token: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Code              int    `json:"code"`
		Msg               string `json:"msg"`
		TenantAccessToken string `json:"tenant_access_token"`
		Expire            int    `json:"expire"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}
	if result.Code != 0 {
		return "", fmt.Errorf("feishu token error: %d %s", result.Code, result.Msg)
	}

	c.accessToken = result.TenantAccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(result.Expire-60) * time.Second)
	return c.accessToken, nil
}

func (c *Client) doAPI(ctx context.Context, method, path string, payload interface{}) (json.RawMessage, error) {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	var bodyReader io.Reader
	if payload != nil {
		data, _ := json.Marshal(payload)
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)

	var envelope struct {
		Code int             `json:"code"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("decode feishu response: %w", err)
	}
	if envelope.Code != 0 {
		return nil, fmt.Errorf("feishu API error: %d %s", envelope.Code, envelope.Msg)
	}
	return envelope.Data, nil
}

// SendTextMessage sends a plain text message to a chat or user.
// receiveIDType: "open_id", "user_id", "union_id", or "chat_id".
func (c *Client) SendTextMessage(ctx context.Context, receiveIDType, receiveID, text string) error {
	content, _ := json.Marshal(map[string]string{"text": text})
	payload := map[string]string{
		"receive_id": receiveID,
		"msg_type":   "text",
		"content":    string(content),
	}
	_, err := c.doAPI(ctx, http.MethodPost, sendMessageURL+"?receive_id_type="+receiveIDType, payload)
	if err != nil {
		slog.Error("[feishu] Send text message failed", "receive_id", receiveID, "error", err)
	}
	return err
}

// SendInteractiveCard sends a rich interactive card message.
func (c *Client) SendInteractiveCard(ctx context.Context, receiveIDType, receiveID string, card *InteractiveCard) error {
	cardJSON, _ := json.Marshal(card)
	payload := map[string]string{
		"receive_id": receiveID,
		"msg_type":   "interactive",
		"content":    string(cardJSON),
	}
	_, err := c.doAPI(ctx, http.MethodPost, sendMessageURL+"?receive_id_type="+receiveIDType, payload)
	if err != nil {
		slog.Error("[feishu] Send card failed", "receive_id", receiveID, "error", err)
	}
	return err
}
