package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	deviceID   string
	httpClient *http.Client
}

func NewClient(baseURL, deviceID string) *Client {
	return &Client{
		baseURL:  baseURL,
		deviceID: deviceID,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type ReportEventRequest struct {
	DeviceID     string                 `json:"device_id"`
	SessionID    string                 `json:"session_id"`
	ActionType   string                 `json:"action_type"`
	Status       string                 `json:"status"`
	Payload      map[string]interface{} `json:"payload"`
	ErrorMessage string                 `json:"error_message"`
}

func (c *Client) ReportEvent(ctx context.Context, actionType, status, sessionID, errMsg string, payload map[string]interface{}) error {
	reqBody := ReportEventRequest{
		DeviceID:     c.deviceID,
		SessionID:    sessionID,
		ActionType:   actionType,
		Status:       status,
		Payload:      payload,
		ErrorMessage: errMsg,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal audit event: %w", err)
	}

	url := fmt.Sprintf("%s/agent/v1/audit-events", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	// TODO: Add Authorization header with DeviceToken

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("audit report request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("audit report failed with status: %d", resp.StatusCode)
	}

	log.Printf("Successfully reported audit event: %s", actionType)
	return nil
}
