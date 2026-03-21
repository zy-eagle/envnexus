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

type WSClient struct {
	serverURL    string
	deviceID     string
	conn         *websocket.Conn
	mu           sync.Mutex
	done         chan struct{}
	registry     *tools.Registry
	auditClient  *audit.Client
	policyEngine *policy.Engine
}

func NewWSClient(serverURL, deviceID string, registry *tools.Registry, auditClient *audit.Client, policyEngine *policy.Engine) *WSClient {
	return &WSClient{
		serverURL:    serverURL,
		deviceID:     deviceID,
		done:         make(chan struct{}),
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
		log.Printf("Invalid WS server URL: %v\n", err)
		return
	}

	// Append device_id to query string
	q := u.Query()
	q.Set("device_id", c.deviceID)
	u.RawQuery = q.Encode()

	dialURL := u.String()
	log.Printf("Connecting to %s\n", dialURL)

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.done:
			return
		default:
		}

		conn, _, err := websocket.DefaultDialer.DialContext(ctx, dialURL, nil)
		if err != nil {
			log.Printf("WS Dial error: %v. Retrying in 5 seconds...\n", err)
			time.Sleep(5 * time.Second)
			continue
		}

		log.Println("Successfully connected to session-gateway")
		c.mu.Lock()
		c.conn = conn
		c.mu.Unlock()

		// Start read and write loops
		errChan := make(chan error, 2)
		go c.readPump(errChan)
		go c.writePump(errChan)

		// Wait for error from either pump
		err = <-errChan
		log.Printf("WS connection lost: %v. Reconnecting...\n", err)

		c.mu.Lock()
		if c.conn != nil {
			c.conn.Close()
			c.conn = nil
		}
		c.mu.Unlock()

		time.Sleep(2 * time.Second) // Small backoff before reconnect
	}
}

type WSMessage struct {
	Type    string                 `json:"type"`
	Payload map[string]interface{} `json:"payload"`
}

func (c *WSClient) readPump(errChan chan<- error) {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()

	if conn == nil {
		return
	}

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			errChan <- fmt.Errorf("read error: %w", err)
			return
		}
		log.Printf("Received message from gateway: %s\n", string(message))
		
		var msg WSMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Failed to unmarshal message: %v\n", err)
			continue
		}

		if msg.Type == "execute_tool" {
			toolName, ok := msg.Payload["tool_name"].(string)
			if !ok {
				log.Println("execute_tool message missing tool_name")
				continue
			}

			tool, found := c.registry.Get(toolName)
			if !found {
				log.Printf("Tool not found: %s\n", toolName)
				continue
			}

			// 1. Policy & Approval Check
			// This will block if the tool requires approval (e.g., flush_dns)
			approved, err := c.policyEngine.Check(context.Background(), tool, msg.Payload)
			if err != nil || !approved {
				log.Printf("Tool execution blocked by policy/approval: %v\n", err)
				
				// Report denial to audit
				go func() {
					_ = c.auditClient.ReportEvent(context.Background(), "tool_execution", "denied", "", err.Error(), msg.Payload)
				}()

				// Send error back via WS
				resultBytes, _ := json.Marshal(map[string]interface{}{
					"type": "tool_result",
					"payload": map[string]interface{}{
						"tool_name": toolName,
						"status": "failed",
						"error": "execution denied or blocked by policy",
					},
				})
				c.SendMessage(websocket.TextMessage, resultBytes)
				continue
			}

			// 2. Execute Tool
			// In a real implementation, we would pass context and handle timeouts
			result, err := tool.Execute(context.Background(), msg.Payload)
			
			// Report Audit Event
			status := "succeeded"
			errMsg := ""
			if err != nil {
				status = "failed"
				errMsg = err.Error()
			} else if result != nil && result.Status == "failed" {
				status = "failed"
				errMsg = result.Error
			}

			// Fire and forget audit report
			go func() {
				auditPayload := map[string]interface{}{
					"tool_name": toolName,
					"params":    msg.Payload,
					"result":    result,
				}
				_ = c.auditClient.ReportEvent(context.Background(), "tool_execution", status, "", errMsg, auditPayload)
			}()

			if err != nil {
				log.Printf("Tool execution failed: %v\n", err)
				continue
			}

			resultBytes, _ := json.Marshal(map[string]interface{}{
				"type": "tool_result",
				"payload": result,
			})
			c.SendMessage(websocket.TextMessage, resultBytes)
		}
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
		case <-ticker.C:
			// Send ping/heartbeat
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

// SendMessage allows other components to send messages through the WS connection
func (c *WSClient) SendMessage(msgType int, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return fmt.Errorf("websocket not connected")
	}

	return c.conn.WriteMessage(msgType, data)
}
