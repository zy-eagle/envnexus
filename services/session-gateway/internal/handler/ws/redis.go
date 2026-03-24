package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	channelAgentEvents   = "enx:agent:events"
	channelSessionEvents = "enx:session:events"
)

type RedisClient struct {
	rdb     *redis.Client
	manager *SessionManager
}

func NewRedisClient(addr, password string, db int) (*RedisClient, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     20,
		MinIdleConns: 5,
	})

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &RedisClient{rdb: rdb}, nil
}

func (rc *RedisClient) SetManager(manager *SessionManager) {
	rc.manager = manager
}

// Publish sends agent-originated events (agent → platform consumer).
func (rc *RedisClient) Publish(evt EventEnvelope) {
	data, err := json.Marshal(evt)
	if err != nil {
		slog.Error("Failed to marshal agent event", "component", "redis", "error", err)
		return
	}
	ctx := context.Background()
	if err := rc.rdb.Publish(ctx, channelAgentEvents, data).Err(); err != nil {
		slog.Error("Failed to publish agent event", "component", "redis", "error", err)
	}
}

// PublishSessionEvent sends platform-originated events for cross-instance delivery
// (platform → gateway instances → agent WebSocket).
func (rc *RedisClient) PublishSessionEvent(evt EventEnvelope) {
	data, err := json.Marshal(evt)
	if err != nil {
		slog.Error("Failed to marshal session event", "component", "redis", "error", err)
		return
	}
	ctx := context.Background()
	if err := rc.rdb.Publish(ctx, channelSessionEvents, data).Err(); err != nil {
		slog.Error("Failed to publish session event", "component", "redis", "error", err)
	}
}

// SubscribeSessionEvents listens for session events on Redis and delivers them to
// locally connected devices. Automatically reconnects on subscription failure.
func (rc *RedisClient) SubscribeSessionEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		rc.subscribeLoop(ctx)

		select {
		case <-ctx.Done():
			return
		case <-time.After(3 * time.Second):
			slog.Info("Reconnecting to session events channel", "component", "redis")
		}
	}
}

func (rc *RedisClient) subscribeLoop(ctx context.Context) {
	sub := rc.rdb.Subscribe(ctx, channelSessionEvents)
	defer sub.Close()

	ch := sub.Channel()
	slog.Info("Subscribed to session events channel", "component", "redis")

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-ch:
			if msg == nil {
				return
			}
			var evt EventEnvelope
			if err := json.Unmarshal([]byte(msg.Payload), &evt); err != nil {
				slog.Warn("Invalid event from session channel", "component", "redis", "error", err)
				continue
			}
			if evt.DeviceID != "" && rc.manager != nil {
				if err := rc.manager.SendToDevice(evt.DeviceID, evt); err != nil {
					slog.Debug("Event not for this instance", "device_id", evt.DeviceID)
				}
			}
		}
	}
}

func (rc *RedisClient) Ping(ctx context.Context) error {
	return rc.rdb.Ping(ctx).Err()
}

func (rc *RedisClient) Close() error {
	return rc.rdb.Close()
}
