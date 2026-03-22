package lifecycle

import (
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

type ConfigResponse struct {
	ConfigVersion string                 `json:"config_version"`
	AgentProfile  map[string]interface{} `json:"agent_profile"`
	PolicyProfile map[string]interface{} `json:"policy_profile"`
}

func (c *Client) Heartbeat(ctx context.Context) error {
	url := fmt.Sprintf("%s/agent/v1/heartbeat", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Authorization", "Bearer "+c.deviceToken)
	req.Header.Set("X-Device-ID", c.deviceID)

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

func (c *Client) GetConfig(ctx context.Context) (*ConfigResponse, error) {
	url := fmt.Sprintf("%s/agent/v1/config", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.deviceToken)
	req.Header.Set("X-Device-ID", c.deviceID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("config request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("config request failed with status: %d", resp.StatusCode)
	}

	var configResp ConfigResponse
	if err := json.NewDecoder(resp.Body).Decode(&configResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &configResp, nil
}
