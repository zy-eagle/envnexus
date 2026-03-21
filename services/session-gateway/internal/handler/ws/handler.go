package ws

import (
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for MVP. In production, validate against allowed origins.
		return true
	},
}

type SessionManager struct {
	// A simple map to hold active connections. Key: DeviceID
	connections map[string]*websocket.Conn
	mu          sync.RWMutex
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		connections: make(map[string]*websocket.Conn),
	}
}

// HandleCommand receives an HTTP request and forwards the JSON payload to the connected agent
func (m *SessionManager) HandleCommand(c *gin.Context) {
	deviceID := c.Query("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing device_id"})
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read body"})
		return
	}
	defer c.Request.Body.Close()

	m.mu.RLock()
	conn, ok := m.connections[deviceID]
	m.mu.RUnlock()

	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Device not connected"})
		return
	}

	// Forward the raw JSON to the agent
	if err := conn.WriteMessage(websocket.TextMessage, body); err != nil {
		log.Printf("Failed to send command to %s: %v\n", deviceID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send command"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "command_sent"})
}

func (m *SessionManager) HandleAgentConnection(c *gin.Context) {
	// 1. Authenticate the request (e.g., via Authorization header or query param)
	deviceID := c.Query("device_id")
	if deviceID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing device_id"})
		return
	}

	// 2. Upgrade HTTP to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v\n", err)
		return
	}

	// 3. Register connection
	m.mu.Lock()
	m.connections[deviceID] = conn
	m.mu.Unlock()

	log.Printf("Device %s connected via WebSocket\n", deviceID)

	// 4. Start read loop
	go m.readPump(deviceID, conn)
}

func (m *SessionManager) readPump(deviceID string, conn *websocket.Conn) {
	defer func() {
		m.mu.Lock()
		delete(m.connections, deviceID)
		m.mu.Unlock()
		conn.Close()
		log.Printf("Device %s disconnected\n", deviceID)
	}()

	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Error reading message from %s: %v\n", deviceID, err)
			}
			break
		}
		
		log.Printf("Received message from %s (type %d): %s\n", deviceID, messageType, string(p))
		
		// MVP: Simple echo back
		if err := conn.WriteMessage(messageType, p); err != nil {
			log.Printf("Error writing message to %s: %v\n", deviceID, err)
			break
		}
	}
}
