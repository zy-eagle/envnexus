package ws

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	HandshakeTimeout: 10 * time.Second,
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
	connections map[string]*DeviceConnection
	mu          sync.RWMutex
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		connections: make(map[string]*DeviceConnection),
	}
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
		return nil
	}

	data, err := json.Marshal(envelope)
	if err != nil {
		return err
	}

	select {
	case dc.SendCh <- data:
		return nil
	default:
		return nil
	}
}

func (m *SessionManager) HandleAgentConnection(c *gin.Context) {
	deviceID := c.Query("device_id")
	if deviceID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing device_id"})
		return
	}

	tenantID := c.Query("tenant_id")

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection for device %s: %v\n", deviceID, err)
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

	log.Printf("Device %s connected (tenant: %s)\n", deviceID, tenantID)

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
				log.Printf("Write error for device %s: %v\n", dc.DeviceID, err)
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
		log.Printf("Device %s disconnected\n", dc.DeviceID)
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
				log.Printf("Read error from device %s: %v\n", dc.DeviceID, err)
			}
			return
		}

		var envelope EventEnvelope
		if err := json.Unmarshal(p, &envelope); err != nil {
			log.Printf("Invalid event from device %s: %v\n", dc.DeviceID, err)
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

	case "session.input", "approval.submit", "session.abort":
		log.Printf("Agent event [%s] from device %s, session %s\n", evt.EventType, dc.DeviceID, evt.SessionID)

	case "tool.completed", "tool.failed":
		log.Printf("Tool event [%s] from device %s: %v\n", evt.EventType, dc.DeviceID, evt.Payload)

	default:
		log.Printf("Unhandled event type [%s] from device %s\n", evt.EventType, dc.DeviceID)
	}
}

func (m *SessionManager) GetOnlineDeviceCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.connections)
}
