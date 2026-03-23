package session

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zy-eagle/envnexus/apps/agent-core/internal/audit"
	"github.com/zy-eagle/envnexus/apps/agent-core/internal/policy"
	"github.com/zy-eagle/envnexus/apps/agent-core/internal/tools"
)

type EventEnvelope struct {
	EventID   string      `json:"event_id"`
	EventType string      `json:"event_type"`
	TenantID  string      `json:"tenant_id,omitempty"`
	DeviceID  string      `json:"device_id,omitempty"`
	SessionID string      `json:"session_id,omitempty"`
	Timestamp string      `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}

type WSClient struct {
	serverURL    string
	deviceID     string
	tenantID     string
	conn         *websocket.Conn
	mu           sync.Mutex
	done         chan struct{}
	sendCh       chan []byte
	registry     *tools.Registry
	auditClient  *audit.Client
	policyEngine *policy.Engine
}

func NewWSClient(serverURL, deviceID, tenantID string, registry *tools.Registry, auditClient *audit.Client, policyEngine *policy.Engine) *WSClient {
	return &WSClient{
		serverURL:    serverURL,
		deviceID:     deviceID,
		tenantID:     tenantID,
		done:         make(chan struct{}),
		sendCh:       make(chan []byte, 64),
		registry:     registry,
		auditClient:  auditClient,
		policyEngine: policyEngine,
	}
}

func (c *WSClient) Start(ctx context.Context) {
	go c.connectLoop(ctx)
}

func (c *WSClient) Stop() {
	close(c.done)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *WSClient) connectLoop(ctx context.Context) {
	u, err := url.Parse(c.serverURL)
	if err != nil {
		log.Printf("[ws] Invalid server URL: %v\n", err)
		return
	}
	q := u.Query()
	q.Set("device_id", c.deviceID)
	q.Set("tenant_id", c.tenantID)
	u.RawQuery = q.Encode()
	dialURL := u.String()

	backoff := time.Second * 2
	maxBackoff := time.Minute

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.done:
			return
		default:
		}

		log.Printf("[ws] Connecting to %s\n", dialURL)
		conn, _, err := websocket.DefaultDialer.DialContext(ctx, dialURL, nil)
		if err != nil {
			log.Printf("[ws] Dial error: %v. Retrying in %v...\n", err, backoff)
			time.Sleep(backoff)
			backoff = backoff * 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		backoff = time.Second * 2
		log.Println("[ws] Connected to session-gateway")

		c.mu.Lock()
		c.conn = conn
		c.mu.Unlock()

		errChan := make(chan error, 2)
		go c.readPump(errChan)
		go c.writePump(errChan)

		err = <-errChan
		log.Printf("[ws] Connection lost: %v. Reconnecting...\n", err)

		c.mu.Lock()
		if c.conn != nil {
			c.conn.Close()
			c.conn = nil
		}
		c.mu.Unlock()

		time.Sleep(time.Second)
	}
}

func (c *WSClient) readPump(errChan chan<- error) {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()

	if conn == nil {
		return
	}

	conn.SetReadDeadline(time.Now().Add(90 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		return nil
	})

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			errChan <- fmt.Errorf("read error: %w", err)
			return
		}

		var evt EventEnvelope
		if err := json.Unmarshal(message, &evt); err != nil {
			log.Printf("[ws] Invalid message: %v\n", err)
			continue
		}

		c.handleServerEvent(evt)
	}
}

func (c *WSClient) writePump(errChan chan<- error) {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()

	if conn == nil {
		return
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.done:
			return
		case msg := <-c.sendCh:
			c.mu.Lock()
			err := conn.WriteMessage(websocket.TextMessage, msg)
			c.mu.Unlock()
			if err != nil {
				errChan <- fmt.Errorf("write error: %w", err)
				return
			}
		case <-ticker.C:
			c.mu.Lock()
			err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second))
			c.mu.Unlock()
			if err != nil {
				errChan <- fmt.Errorf("ping error: %w", err)
				return
			}
		}
	}
}

func (c *WSClient) handleServerEvent(evt EventEnvelope) {
	switch evt.EventType {
	case "execute_tool":
		c.handleExecuteTool(evt)
	case "heartbeat.pong":
		log.Println("[ws] Heartbeat pong received")
	case "session.start":
		log.Printf("[ws] Session started: %s\n", evt.SessionID)
	case "session.end":
		log.Printf("[ws] Session ended: %s\n", evt.SessionID)
	default:
		log.Printf("[ws] Unhandled event: %s\n", evt.EventType)
	}
}

func (c *WSClient) handleExecuteTool(evt EventEnvelope) {
	payloadMap, ok := evt.Payload.(map[string]interface{})
	if !ok {
		return
	}
	toolName, _ := payloadMap["tool_name"].(string)
	if toolName == "" {
		return
	}

	tool, found := c.registry.Get(toolName)
	if !found {
		c.sendToolResult(evt.SessionID, toolName, nil, fmt.Errorf("tool not found: %s", toolName))
		return
	}

	approved, err := c.policyEngine.Check(context.Background(), tool, payloadMap)
	if err != nil || !approved {
		c.auditClient.ReportEvent(context.Background(), "tool.denied", evt.SessionID, map[string]interface{}{
			"tool_name": toolName,
			"reason":    fmt.Sprintf("%v", err),
		})
		c.sendToolResult(evt.SessionID, toolName, nil, fmt.Errorf("denied by policy"))
		return
	}

	result, err := tool.Execute(context.Background(), payloadMap)

	c.auditClient.ReportEvent(context.Background(), "tool.completed", evt.SessionID, map[string]interface{}{
		"tool_name": toolName,
		"result":    result,
	})

	c.sendToolResult(evt.SessionID, toolName, result, err)
}

func (c *WSClient) sendToolResult(sessionID, toolName string, result *tools.ToolResult, err error) {
	payload := map[string]interface{}{
		"tool_name": toolName,
	}
	evtType := "tool.completed"

	if err != nil {
		evtType = "tool.failed"
		payload["status"] = "failed"
		payload["error"] = err.Error()
	} else if result != nil {
		payload["status"] = result.Status
		payload["output"] = result.Output
		payload["summary"] = result.Summary
		payload["duration_ms"] = result.DurationMs
	}

	envelope := EventEnvelope{
		EventType: evtType,
		DeviceID:  c.deviceID,
		SessionID: sessionID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Payload:   payload,
	}

	data, _ := json.Marshal(envelope)
	select {
	case c.sendCh <- data:
	default:
		log.Println("[ws] Send buffer full, dropping tool result")
	}
}

func (c *WSClient) SendEvent(evt EventEnvelope) {
	data, _ := json.Marshal(evt)
	select {
	case c.sendCh <- data:
	default:
	}
}
