package lifecycle

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	baseURL     string
	deviceID    string
	deviceToken string
	httpClient  *http.Client
}

func NewClient(baseURL, deviceID, deviceToken string) *Client {
	return &Client{
		baseURL:     baseURL,
		deviceID:    deviceID,
		deviceToken: deviceToken,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type HeartbeatRequest struct {
	DeviceID      string `json:"device_id"`
	Status        string `json:"status"`
	AgentVersion  string `json:"agent_version"`
	PolicyVersion int    `json:"policy_version"`
}

type ConfigResponse struct {
	HasUpdate     bool            `json:"has_update"`
	ConfigVersion int             `json:"config_version"`
	AgentProfile  json.RawMessage `json:"agent_profile"`
	ModelProfile  json.RawMessage `json:"model_profile"`
	PolicyProfile json.RawMessage `json:"policy_profile"`
}

func (c *Client) Heartbeat(ctx context.Context, agentVersion string, policyVersion int) error {
	reqBody := HeartbeatRequest{
		DeviceID:      c.deviceID,
		Status:        "active",
		AgentVersion:  agentVersion,
		PolicyVersion: policyVersion,
	}
	jsonData, _ := json.Marshal(reqBody)

	url := fmt.Sprintf("%s/agent/v1/heartbeat", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.deviceToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("heartbeat request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("heartbeat failed with status: %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) GetSessionToken(ctx context.Context, deviceID, tenantID string) (string, error) {
	reqBody := map[string]string{
		"device_id":      deviceID,
		"transport":      "websocket",
		"initiator_type": "agent",
	}
	jsonData, _ := json.Marshal(reqBody)

	url := fmt.Sprintf("%s/agent/v1/sessions", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create session request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.deviceToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("session request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("session request failed with status: %d", resp.StatusCode)
	}

	var apiResp struct {
		Data struct {
			WSToken string `json:"ws_token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return "", fmt.Errorf("failed to decode session response: %w", err)
	}

	return apiResp.Data.WSToken, nil
}

func (c *Client) GetConfig(ctx context.Context, currentConfigVersion int) (*ConfigResponse, error) {
	url := fmt.Sprintf("%s/agent/v1/config?device_id=%s&current_config_version=%d",
		c.baseURL, c.deviceID, currentConfigVersion)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.deviceToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("config request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("config request failed with status: %d", resp.StatusCode)
	}

	var apiResp struct {
		Data ConfigResponse `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &apiResp.Data, nil
}
