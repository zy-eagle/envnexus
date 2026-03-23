package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zy-eagle/envnexus/apps/agent-core/internal/audit"
	"github.com/zy-eagle/envnexus/apps/agent-core/internal/diagnosis"
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
	deviceToken  string
	conn         *websocket.Conn
	mu           sync.Mutex
	done         chan struct{}
	sendCh       chan []byte
	registry     *tools.Registry
	auditClient  *audit.Client
	policyEngine *policy.Engine
	diagEngine   *diagnosis.Engine
}

func NewWSClient(serverURL, deviceID, tenantID, deviceToken string, registry *tools.Registry, auditClient *audit.Client, policyEngine *policy.Engine, diagEngine *diagnosis.Engine) *WSClient {
	return &WSClient{
		serverURL:    serverURL,
		deviceID:     deviceID,
		tenantID:     tenantID,
		deviceToken:  deviceToken,
		done:         make(chan struct{}),
		sendCh:       make(chan []byte, 64),
		registry:     registry,
		auditClient:  auditClient,
		policyEngine: policyEngine,
		diagEngine:   diagEngine,
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
	dialURL := c.buildDialURL()
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

		slog.Info("[ws] Connecting to session-gateway", "dial_url", dialURL)
		conn, _, err := websocket.DefaultDialer.DialContext(ctx, dialURL, nil)
		if err != nil {
			slog.Warn("[ws] Dial error, retrying", "error", err, "backoff", backoff)
			time.Sleep(backoff)
			backoff = backoff * 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		backoff = time.Second * 2
		slog.Info("[ws] Connected to session-gateway")

		c.mu.Lock()
		c.conn = conn
		c.mu.Unlock()

		errChan := make(chan error, 2)
		go c.readPump(errChan)
		go c.writePump(errChan)

		err = <-errChan
		slog.Warn("[ws] Connection lost, reconnecting", "error", err)

		c.mu.Lock()
		if c.conn != nil {
			c.conn.Close()
			c.conn = nil
		}
		c.mu.Unlock()

		time.Sleep(time.Second)
	}
}

func (c *WSClient) buildDialURL() string {
	u, err := url.Parse(c.serverURL)
	if err != nil {
		slog.Warn("[ws] Invalid server URL, using as-is", "error", err)
		return c.serverURL
	}

	if u.Path == "" || u.Path == "/" {
		u.Path = "/ws/v1/sessions/" + c.deviceID
	}

	q := u.Query()
	if c.deviceToken != "" {
		q.Set("token", c.deviceToken)
	}
	q.Set("tenant_id", c.tenantID)
	u.RawQuery = q.Encode()
	return u.String()
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
			slog.Warn("[ws] Invalid message", "error", err)
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
	case "session.created":
		slog.Info("[ws] Session created", "session_id", evt.SessionID)
		c.handleSessionCreated(evt)

	case "diagnosis.started":
		slog.Info("[ws] Diagnosis started", "session_id", evt.SessionID)

	case "diagnosis.completed":
		slog.Info("[ws] Diagnosis completed", "session_id", evt.SessionID)

	case "approval.requested":
		slog.Info("[ws] Approval requested", "session_id", evt.SessionID)
		c.handleApprovalRequested(evt)

	case "approval.expired":
		slog.Warn("[ws] Approval expired", "session_id", evt.SessionID)

	case "tool.started":
		slog.Info("[ws] Tool started", "session_id", evt.SessionID)
		c.handleToolStarted(evt)

	case "tool.completed":
		slog.Info("[ws] Tool completed", "session_id", evt.SessionID)

	case "session.completed":
		slog.Info("[ws] Session completed", "session_id", evt.SessionID)

	case "heartbeat.pong":
		slog.Info("[ws] Heartbeat pong received")

	default:
		slog.Info("[ws] Unhandled event", "event_type", evt.EventType)
	}
}

func (c *WSClient) handleSessionCreated(evt EventEnvelope) {
	payloadMap, ok := evt.Payload.(map[string]interface{})
	if !ok {
		return
	}
	initialMessage, _ := payloadMap["initial_message"].(string)
	if initialMessage == "" {
		return
	}

	c.SendEvent(EventEnvelope{
		EventID:   generateEventID(),
		EventType: "diagnosis.started",
		DeviceID:  c.deviceID,
		SessionID: evt.SessionID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Payload:   map[string]interface{}{"intent": initialMessage},
	})

	go func() {
		result, err := c.diagEngine.RunDiagnosis(context.Background(), evt.SessionID, initialMessage)
		if err != nil {
			slog.Error("[ws] Diagnosis failed", "session_id", evt.SessionID, "error", err)
			c.SendEvent(EventEnvelope{
				EventID:   generateEventID(),
				EventType: "diagnosis.completed",
				DeviceID:  c.deviceID,
				SessionID: evt.SessionID,
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				Payload:   map[string]interface{}{"error": err.Error()},
			})
			return
		}

		c.SendEvent(EventEnvelope{
			EventID:   generateEventID(),
			EventType: "diagnosis.completed",
			DeviceID:  c.deviceID,
			SessionID: evt.SessionID,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Payload:   result,
		})

		if result.ApprovalRequired && len(result.RecommendedActions) > 0 {
			for _, action := range result.RecommendedActions {
				c.SendEvent(EventEnvelope{
					EventID:   generateEventID(),
					EventType: "approval.requested",
					DeviceID:  c.deviceID,
					SessionID: evt.SessionID,
					Timestamp: time.Now().UTC().Format(time.RFC3339),
					Payload: map[string]interface{}{
						"tool_name":   action.ToolName,
						"description": action.Description,
						"risk_level":  action.RiskLevel,
						"params":      action.Params,
					},
				})
			}
		}

		c.auditClient.ReportEvent(context.Background(), "diagnosis.completed", evt.SessionID, map[string]interface{}{
			"problem_type": result.ProblemType,
			"confidence":   result.Confidence,
			"findings":     len(result.Findings),
			"actions":      len(result.RecommendedActions),
		})
	}()
}

func (c *WSClient) handleToolStarted(evt EventEnvelope) {
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

	params, _ := payloadMap["params"].(map[string]interface{})

	approved, err := c.policyEngine.Check(context.Background(), tool, params)
	if err != nil || !approved {
		c.auditClient.ReportEvent(context.Background(), "tool.denied", evt.SessionID, map[string]interface{}{
			"tool_name": toolName,
			"reason":    fmt.Sprintf("%v", err),
		})
		c.sendToolResult(evt.SessionID, toolName, nil, fmt.Errorf("denied by policy"))
		return
	}

	result, err := tool.Execute(context.Background(), params)

	eventType := "tool.completed"
	if err != nil || (result != nil && result.Status == "failed") {
		eventType = "tool.failed"
	}

	c.auditClient.ReportEvent(context.Background(), eventType, evt.SessionID, map[string]interface{}{
		"tool_name": toolName,
		"result":    result,
	})

	c.sendToolResult(evt.SessionID, toolName, result, err)
}

func (c *WSClient) handleApprovalRequested(evt EventEnvelope) {
	c.auditClient.ReportEvent(context.Background(), "approval.requested", evt.SessionID, evt.Payload)
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
		EventID:   generateEventID(),
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
		slog.Warn("[ws] Send buffer full, dropping tool result")
	}
}

func (c *WSClient) SendEvent(evt EventEnvelope) {
	if evt.EventID == "" {
		evt.EventID = generateEventID()
	}
	if evt.Timestamp == "" {
		evt.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	data, _ := json.Marshal(evt)
	select {
	case c.sendCh <- data:
	default:
	}
}

func generateEventID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
