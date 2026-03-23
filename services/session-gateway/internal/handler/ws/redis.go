package ws

import (
	"context"
	"encoding/json"
	"log"

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
		Addr:     addr,
		Password: password,
		DB:       db,
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

func (rc *RedisClient) Publish(evt EventEnvelope) {
	data, err := json.Marshal(evt)
	if err != nil {
		log.Printf("[redis] Failed to marshal event: %v", err)
		return
	}
	ctx := context.Background()
	if err := rc.rdb.Publish(ctx, channelAgentEvents, data).Err(); err != nil {
		log.Printf("[redis] Failed to publish event: %v", err)
	}
}

func (rc *RedisClient) SubscribeSessionEvents(ctx context.Context) {
	sub := rc.rdb.Subscribe(ctx, channelSessionEvents)
	defer sub.Close()

	ch := sub.Channel()
	log.Println("[redis] Subscribed to session events channel")

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
				log.Printf("[redis] Invalid event from channel: %v", err)
				continue
			}
			if evt.DeviceID != "" && rc.manager != nil {
				if err := rc.manager.SendToDevice(evt.DeviceID, evt); err != nil {
					log.Printf("[redis] Failed to forward event to device %s: %v", evt.DeviceID, err)
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
