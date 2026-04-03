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
	baseURL     string
	httpClient  *http.Client
	redisClient *RedisClient
}

func NewGatewayClient(baseURL string) *GatewayClient {
	return &GatewayClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// SetRedisClient enables Redis pub/sub fallback for cross-instance event delivery.
func (g *GatewayClient) SetRedisClient(rc *RedisClient) {
	g.redisClient = rc
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

	if err := g.sendEvent(ctx, sessionID, evt); err != nil {
		slog.Warn("HTTP delivery to gateway failed, trying Redis pub/sub", "error", err)
		return g.publishViaRedis(ctx, evt)
	}
	return nil
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

// SendToDevice sends an event to a specific device via the session gateway.
func (g *GatewayClient) SendToDevice(ctx context.Context, deviceID string, evt SessionEvent) error {
	body, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	url := fmt.Sprintf("%s/api/v1/devices/%s/events", g.baseURL, deviceID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := g.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send event to device: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("gateway returned status %d for device %s", resp.StatusCode, deviceID)
	}
	return nil
}

func (g *GatewayClient) publishViaRedis(ctx context.Context, evt SessionEvent) error {
	if g.redisClient == nil {
		return fmt.Errorf("no Redis client available for pub/sub fallback")
	}
	data, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("marshal event for Redis: %w", err)
	}
	return g.redisClient.Publish(ctx, "enx:session:events", string(data))
}
