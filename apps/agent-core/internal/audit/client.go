package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

type Client struct {
	baseURL     string
	deviceID    string
	deviceToken string
	httpClient  *http.Client

	mu    sync.Mutex
	queue []EventItem
}

type EventItem struct {
	EventType    string      `json:"event_type"`
	SessionID    string      `json:"session_id,omitempty"`
	EventPayload interface{} `json:"event_payload"`
}

type BatchRequest struct {
	Events []EventItem `json:"events"`
}

func NewClient(baseURL, deviceID, deviceToken string) *Client {
	return &Client{
		baseURL:     baseURL,
		deviceID:    deviceID,
		deviceToken: deviceToken,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		queue: make([]EventItem, 0, 32),
	}
}

func (c *Client) Enqueue(eventType, sessionID string, payload interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.queue = append(c.queue, EventItem{
		EventType:    eventType,
		SessionID:    sessionID,
		EventPayload: payload,
	})
}

func (c *Client) StartFlushLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.Flush(context.Background())
			return
		case <-ticker.C:
			c.Flush(ctx)
		}
	}
}

func (c *Client) Flush(ctx context.Context) {
	c.mu.Lock()
	if len(c.queue) == 0 {
		c.mu.Unlock()
		return
	}
	batch := make([]EventItem, len(c.queue))
	copy(batch, c.queue)
	c.queue = c.queue[:0]
	c.mu.Unlock()

	reqBody := BatchRequest{Events: batch}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		slog.Error("[audit] Failed to marshal batch", "error", err)
		return
	}

	url := fmt.Sprintf("%s/agent/v1/audit-events", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		slog.Error("[audit] Failed to create request", "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if c.deviceToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.deviceToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		slog.Error("[audit] Batch flush failed", "error", err)
		c.mu.Lock()
		c.queue = append(batch, c.queue...)
		c.mu.Unlock()
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		slog.Warn("[audit] Batch flush unexpected status", "status", resp.StatusCode)
	} else {
		slog.Info("[audit] Flushed events", "count", len(batch))
	}
}

func (c *Client) ReportEvent(ctx context.Context, eventType, sessionID string, payload interface{}) {
	c.Enqueue(eventType, sessionID, payload)
}
