package enrollment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

type EnrollRequest struct {
	EnrollmentToken string     `json:"enrollment_token"`
	Device          DeviceInfo `json:"device"`
	Agent           AgentInfo  `json:"agent"`
}

type DeviceInfo struct {
	DeviceName      string `json:"device_name"`
	Hostname        string `json:"hostname"`
	Platform        string `json:"platform"`
	Arch            string `json:"arch"`
	EnvironmentType string `json:"environment_type"`
}

type AgentInfo struct {
	Version string `json:"version"`
}

type EnrollResponse struct {
	DeviceID      string `json:"device_id"`
	TenantID      string `json:"tenant_id"`
	DeviceToken   string `json:"device_token"`
	ConfigVersion int    `json:"config_version"`
}

type APIResponse struct {
	Data  json.RawMessage `json:"data"`
	Error *APIError       `json:"error"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (c *Client) Enroll(ctx context.Context, enrollmentToken, agentVersion string) (*EnrollResponse, error) {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown-host"
	}

	reqBody := EnrollRequest{
		EnrollmentToken: enrollmentToken,
		Device: DeviceInfo{
			DeviceName:      hostname,
			Hostname:        hostname,
			Platform:        runtime.GOOS,
			Arch:            runtime.GOARCH,
			EnvironmentType: "physical",
		},
		Agent: AgentInfo{
			Version: agentVersion,
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal enroll request: %w", err)
	}

	url := fmt.Sprintf("%s/agent/v1/enroll", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("enroll request failed: %w", err)
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if apiResp.Error != nil {
		return nil, fmt.Errorf("enrollment failed: [%s] %s", apiResp.Error.Code, apiResp.Error.Message)
	}

	var enrollResp EnrollResponse
	if err := json.Unmarshal(apiResp.Data, &enrollResp); err != nil {
		return nil, fmt.Errorf("failed to parse enrollment data: %w", err)
	}

	return &enrollResp, nil
}
