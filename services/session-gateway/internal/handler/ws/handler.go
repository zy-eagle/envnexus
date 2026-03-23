package ws

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func newUpgrader(allowedOrigins []string) websocket.Upgrader {
	allowed := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = true
	}
	return websocket.Upgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		CheckOrigin: func(r *http.Request) bool {
			if len(allowed) == 0 {
				return false
			}
			origin := r.Header.Get("Origin")
			return allowed[origin]
		},
		HandshakeTimeout: 10 * time.Second,
	}
}

type EventEnvelope struct {
	EventID   string      `json:"event_id"`
	EventType string      `json:"event_type"`
	TenantID  string      `json:"tenant_id,omitempty"`
	DeviceID  string      `json:"device_id,omitempty"`
	SessionID string      `json:"session_id,omitempty"`
	Timestamp string      `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}

type DeviceConnection struct {
	Conn     *websocket.Conn
	DeviceID string
	TenantID string
	SendCh   chan []byte
}

type SessionManager struct {
	connections    map[string]*DeviceConnection
	mu             sync.RWMutex
	tokenSecret    string
	redisClient    *RedisClient
	upgrader       websocket.Upgrader
}

func NewSessionManager(tokenSecret string, allowedOrigins []string) *SessionManager {
	return &SessionManager{
		connections: make(map[string]*DeviceConnection),
		tokenSecret: tokenSecret,
		upgrader:    newUpgrader(allowedOrigins),
	}
}

func (m *SessionManager) SetRedisClient(rc *RedisClient) {
	m.redisClient = rc
	rc.SetManager(m)
}

func (m *SessionManager) HandleCommand(c *gin.Context) {
	deviceID := c.Query("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing device_id"})
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read body"})
		return
	}
	defer c.Request.Body.Close()

	m.mu.RLock()
	dc, ok := m.connections[deviceID]
	m.mu.RUnlock()

	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not connected"})
		return
	}

	select {
	case dc.SendCh <- body:
		c.JSON(http.StatusOK, gin.H{"status": "command_sent"})
	default:
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "device send buffer full"})
	}
}

func (m *SessionManager) SendToDevice(deviceID string, envelope EventEnvelope) error {
	m.mu.RLock()
	dc, ok := m.connections[deviceID]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("device %s not connected", deviceID)
	}

	data, err := json.Marshal(envelope)
	if err != nil {
		return err
	}

	select {
	case dc.SendCh <- data:
		return nil
	default:
		return fmt.Errorf("send buffer full for device %s", deviceID)
	}
}

func (m *SessionManager) BroadcastToTenant(tenantID string, envelope EventEnvelope) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, err := json.Marshal(envelope)
	if err != nil {
		return
	}

	for _, dc := range m.connections {
		if dc.TenantID == tenantID {
			select {
			case dc.SendCh <- data:
			default:
			}
		}
	}
}

// POST /api/v1/sessions/:sessionId/events — platform-api dispatches events to agents
func (m *SessionManager) HandleSessionEvent(c *gin.Context) {
	var envelope EventEnvelope
	if err := c.ShouldBindJSON(&envelope); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	deviceID := envelope.DeviceID
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing device_id in envelope"})
		return
	}

	if err := m.SendToDevice(deviceID, envelope); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "delivered"})
}

// GET /ws/v1/sessions/:deviceId?token=... — agent connects with session token
func (m *SessionManager) HandleAgentConnection(c *gin.Context) {
	deviceID := c.Param("deviceId")
	if deviceID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing device_id"})
		return
	}

	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing session token"})
		return
	}

	claims, err := ValidateSessionToken(token, m.tokenSecret)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid session token"})
		return
	}
	if claims.DeviceID != deviceID {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token device mismatch"})
		return
	}
	tenantID := claims.TenantID

	conn, err := m.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		slog.Info("Failed to upgrade connection", "device_id", deviceID, "error", err)
		return
	}

	dc := &DeviceConnection{
		Conn:     conn,
		DeviceID: deviceID,
		TenantID: tenantID,
		SendCh:   make(chan []byte, 64),
	}

	m.mu.Lock()
	if old, exists := m.connections[deviceID]; exists {
		close(old.SendCh)
		old.Conn.Close()
	}
	m.connections[deviceID] = dc
	m.mu.Unlock()

	slog.Info("Device connected", "device_id", deviceID, "tenant_id", tenantID)

	go m.writePump(dc)
	go m.readPump(dc)
}

func (m *SessionManager) writePump(dc *DeviceConnection) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		dc.Conn.Close()
	}()

	for {
		select {
		case msg, ok := <-dc.SendCh:
			if !ok {
				_ = dc.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			_ = dc.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := dc.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				slog.Info("Write error for device", "device_id", dc.DeviceID, "error", err)
				return
			}
		case <-ticker.C:
			_ = dc.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := dc.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (m *SessionManager) readPump(dc *DeviceConnection) {
	defer func() {
		m.mu.Lock()
		if current, ok := m.connections[dc.DeviceID]; ok && current == dc {
			delete(m.connections, dc.DeviceID)
		}
		m.mu.Unlock()
		close(dc.SendCh)
		dc.Conn.Close()
		slog.Info("Device disconnected", "device_id", dc.DeviceID)
	}()

	dc.Conn.SetReadDeadline(time.Now().Add(90 * time.Second))
	dc.Conn.SetPongHandler(func(string) error {
		dc.Conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		return nil
	})

	for {
		_, p, err := dc.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Info("Read error from device", "device_id", dc.DeviceID, "error", err)
			}
			return
		}

		var envelope EventEnvelope
		if err := json.Unmarshal(p, &envelope); err != nil {
			slog.Info("Invalid event from device", "device_id", dc.DeviceID, "error", err)
			continue
		}

		envelope.DeviceID = dc.DeviceID
		envelope.TenantID = dc.TenantID

		m.handleAgentEvent(dc, envelope)
	}
}

func (m *SessionManager) handleAgentEvent(dc *DeviceConnection, evt EventEnvelope) {
	switch evt.EventType {
	case "heartbeat.ping":
		ack := EventEnvelope{
			EventID:   evt.EventID,
			EventType: "heartbeat.pong",
			DeviceID:  dc.DeviceID,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
		data, _ := json.Marshal(ack)
		select {
		case dc.SendCh <- data:
		default:
		}

	case "session.input":
		slog.Info("Session input from device", "device_id", dc.DeviceID, "session_id", evt.SessionID)
		m.publishToRedis(evt)

	case "approval.submit":
		slog.Info("Approval submit from device", "device_id", dc.DeviceID, "session_id", evt.SessionID)
		m.publishToRedis(evt)

	case "session.abort":
		slog.Info("Session abort from device", "device_id", dc.DeviceID, "session_id", evt.SessionID)
		m.publishToRedis(evt)

	case "tool.completed":
		slog.Info("Tool completed from device", "device_id", dc.DeviceID, "session_id", evt.SessionID)
		m.publishToRedis(evt)

	case "tool.failed":
		slog.Info("Tool failed from device", "device_id", dc.DeviceID, "session_id", evt.SessionID)
		m.publishToRedis(evt)

	case "diagnosis.started":
		slog.Info("Diagnosis started from device", "device_id", dc.DeviceID, "session_id", evt.SessionID)
		m.publishToRedis(evt)

	case "diagnosis.completed":
		slog.Info("Diagnosis completed from device", "device_id", dc.DeviceID, "session_id", evt.SessionID)
		m.publishToRedis(evt)

	case "approval.requested":
		slog.Info("Approval requested from device", "device_id", dc.DeviceID, "session_id", evt.SessionID)
		m.publishToRedis(evt)

	default:
		slog.Info("Unhandled event type from device", "event_type", evt.EventType, "device_id", dc.DeviceID)
	}
}

func (m *SessionManager) publishToRedis(evt EventEnvelope) {
	if m.redisClient != nil {
		m.redisClient.Publish(evt)
	}
}

func (m *SessionManager) GetOnlineDeviceCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.connections)
}

func (m *SessionManager) GetOnlineDevices() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	devices := make([]string, 0, len(m.connections))
	for id := range m.connections {
		devices = append(devices, id)
	}
	return devices
}

func (m *SessionManager) IsDeviceOnline(deviceID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.connections[deviceID]
	return ok
}
