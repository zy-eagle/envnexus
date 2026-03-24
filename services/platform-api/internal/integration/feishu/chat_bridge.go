package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// ChatBinding stores the mapping between a Feishu chat and an EnvNexus device.
type ChatBinding struct {
	ChatID    string    `json:"chat_id"`
	DeviceID  string    `json:"device_id"`
	TenantID  string    `json:"tenant_id"`
	BoundBy   string    `json:"bound_by"`
	BoundAt   time.Time `json:"bound_at"`
	SessionID string    `json:"session_id,omitempty"`
}

// RedisStore is the interface ChatBridge needs from the Redis client.
type RedisStore interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Del(ctx context.Context, keys ...string) error
}

// ChatBridge manages bidirectional mappings between Feishu chats and EnvNexus
// devices/sessions, enabling conversational interaction.
type ChatBridge struct {
	mu       sync.RWMutex
	bindings map[string]*ChatBinding // chat_id -> binding
	sessions map[string]string       // session_id -> chat_id (reverse index)
	redis    RedisStore
}

const redisBindingPrefix = "enx:feishu:bind:"
const redisSessionPrefix = "enx:feishu:sess:"

func NewChatBridge(redis RedisStore) *ChatBridge {
	return &ChatBridge{
		bindings: make(map[string]*ChatBinding),
		sessions: make(map[string]string),
		redis:    redis,
	}
}

// Bind associates a Feishu chat_id with a device.
func (cb *ChatBridge) Bind(ctx context.Context, chatID, deviceID, tenantID, userID string) *ChatBinding {
	binding := &ChatBinding{
		ChatID:   chatID,
		DeviceID: deviceID,
		TenantID: tenantID,
		BoundBy:  userID,
		BoundAt:  time.Now(),
	}

	cb.mu.Lock()
	cb.bindings[chatID] = binding
	cb.mu.Unlock()

	if cb.redis != nil {
		data, _ := json.Marshal(binding)
		if err := cb.redis.Set(ctx, redisBindingPrefix+chatID, string(data), 0); err != nil {
			slog.Warn("[feishu-bridge] Redis set failed", "chat_id", chatID, "error", err)
		}
	}

	slog.Info("[feishu-bridge] Chat bound to device", "chat_id", chatID, "device_id", deviceID)
	return binding
}

// Unbind removes the association for a Feishu chat.
func (cb *ChatBridge) Unbind(ctx context.Context, chatID string) {
	cb.mu.Lock()
	binding := cb.bindings[chatID]
	if binding != nil && binding.SessionID != "" {
		delete(cb.sessions, binding.SessionID)
	}
	delete(cb.bindings, chatID)
	cb.mu.Unlock()

	if cb.redis != nil {
		_ = cb.redis.Del(ctx, redisBindingPrefix+chatID)
	}
	slog.Info("[feishu-bridge] Chat unbound", "chat_id", chatID)
}

// GetBinding returns the binding for a chat, checking Redis if not in memory.
func (cb *ChatBridge) GetBinding(ctx context.Context, chatID string) *ChatBinding {
	cb.mu.RLock()
	b := cb.bindings[chatID]
	cb.mu.RUnlock()
	if b != nil {
		return b
	}

	if cb.redis == nil {
		return nil
	}
	val, err := cb.redis.Get(ctx, redisBindingPrefix+chatID)
	if err != nil || val == "" {
		return nil
	}
	var binding ChatBinding
	if json.Unmarshal([]byte(val), &binding) != nil {
		return nil
	}

	cb.mu.Lock()
	cb.bindings[chatID] = &binding
	cb.mu.Unlock()
	return &binding
}

// TrackSession records that a session belongs to a chat.
func (cb *ChatBridge) TrackSession(ctx context.Context, chatID, sessionID string) {
	cb.mu.Lock()
	cb.sessions[sessionID] = chatID
	if b := cb.bindings[chatID]; b != nil {
		b.SessionID = sessionID
	}
	cb.mu.Unlock()

	if cb.redis != nil {
		_ = cb.redis.Set(ctx, redisSessionPrefix+sessionID, chatID, 2*time.Hour)
	}
}

// ChatForSession returns the chat_id associated with a session.
func (cb *ChatBridge) ChatForSession(ctx context.Context, sessionID string) string {
	cb.mu.RLock()
	chatID := cb.sessions[sessionID]
	cb.mu.RUnlock()
	if chatID != "" {
		return chatID
	}

	if cb.redis == nil {
		return ""
	}
	val, err := cb.redis.Get(ctx, redisSessionPrefix+sessionID)
	if err != nil || val == "" {
		return ""
	}

	cb.mu.Lock()
	cb.sessions[sessionID] = val
	cb.mu.Unlock()
	return val
}

// ChatForDevice returns the first chat_id bound to a specific device.
func (cb *ChatBridge) ChatForDevice(ctx context.Context, deviceID string) string {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	for _, b := range cb.bindings {
		if b.DeviceID == deviceID {
			return b.ChatID
		}
	}
	return ""
}

// FormatBindingInfo returns a human-readable summary of a binding.
func FormatBindingInfo(b *ChatBinding) string {
	elapsed := time.Since(b.BoundAt).Truncate(time.Second)
	session := "无"
	if b.SessionID != "" {
		session = b.SessionID
	}
	return fmt.Sprintf(
		"📎 **当前绑定信息**\n\n"+
			"**设备**: `%s`\n"+
			"**租户**: `%s`\n"+
			"**绑定人**: %s\n"+
			"**已绑定**: %s\n"+
			"**当前会话**: %s",
		b.DeviceID, b.TenantID, b.BoundBy, elapsed.String(), session,
	)
}
