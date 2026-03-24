package infrastructure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

type GatewayClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewGatewayClient(baseURL string) *GatewayClient {
	return &GatewayClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

type SessionEvent struct {
	EventID   string      `json:"event_id"`
	EventType string      `json:"event_type"`
	TenantID  string      `json:"tenant_id"`
	DeviceID  string      `json:"device_id"`
	SessionID string      `json:"session_id"`
	Timestamp string      `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}

func (g *GatewayClient) NotifySessionCreated(ctx context.Context, tenantID, deviceID, sessionID string) error {
	evt := SessionEvent{
		EventID:   fmt.Sprintf("evt_%d", time.Now().UnixNano()),
		EventType: "session.created",
		TenantID:  tenantID,
		DeviceID:  deviceID,
		SessionID: sessionID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Payload: map[string]string{
			"session_id": sessionID,
			"device_id":  deviceID,
		},
	}

	return g.sendEvent(ctx, sessionID, evt)
}

func (g *GatewayClient) sendEvent(ctx context.Context, sessionID string, evt SessionEvent) error {
	body, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/sessions/%s/events", g.baseURL, sessionID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send event to gateway: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		slog.Warn("gateway returned error", "status", resp.StatusCode, "session_id", sessionID)
	}
	return nil
}
