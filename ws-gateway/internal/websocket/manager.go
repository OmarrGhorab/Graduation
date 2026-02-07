package websocket

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/graduation/ws-gateway/internal/events"
	"github.com/redis/go-redis/v9"
)

type Manager struct {
	Clients    map[string][]*Client // UserID -> List of Connections (Tabs)
	Register   chan *Client
	Unregister chan *Client
	Redis      *redis.Client
	mu         sync.RWMutex
}

func NewManager(redisClient *redis.Client) *Manager {
	return &Manager{
		Clients:    make(map[string][]*Client),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Redis:      redisClient,
	}
}

func (m *Manager) Run() {
	// Start Redis Subscriber
	go m.subscribeToRedis()

	for {
		select {
		case client := <-m.Register:
			m.mu.Lock()
			wasOffline := len(m.Clients[client.UserID]) == 0
			m.Clients[client.UserID] = append(m.Clients[client.UserID], client)
			m.mu.Unlock()
			log.Printf("User registered: %s", client.UserID)

			// If this is the first connection, mark user as online
			if wasOffline {
				m.setUserOnline(client.UserID, true)
			}

		case client := <-m.Unregister:
			m.mu.Lock()
			if clients, ok := m.Clients[client.UserID]; ok {
				// Remove specific client
				for i, c := range clients {
					if c == client {
						m.Clients[client.UserID] = append(clients[:i], clients[i+1:]...)
						break
					}
				}
				// If no more connections, mark user as offline
				if len(m.Clients[client.UserID]) == 0 {
					delete(m.Clients, client.UserID)
					m.mu.Unlock()
					m.setUserOnline(client.UserID, false)
				} else {
					m.mu.Unlock()
				}
			} else {
				m.mu.Unlock()
			}
			close(client.Send)
		}
	}
}

func (m *Manager) subscribeToRedis() {
	ctx := context.Background()
	pubsub := m.Redis.Subscribe(ctx, "chat.live.updates")
	defer pubsub.Close()

	ch := pubsub.Channel()
	for msg := range ch {
		var envelope events.RedisEnvelope
		if err := json.Unmarshal([]byte(msg.Payload), &envelope); err != nil {
			log.Printf("Error unmarshalling redis update: %v", err)
			continue
		}

		// Handle presence events differently - broadcast to all
		if envelope.Type == events.TopicUserPresence {
			m.mu.RLock()
			wsPayload := CreateWSPayload(envelope.Type, envelope.Payload)
			for _, clients := range m.Clients {
				for _, client := range clients {
					select {
					case client.Send <- wsPayload:
					default:
						close(client.Send)
					}
				}
			}
			m.mu.RUnlock()
			continue
		}

		// Find local Targets for other events
		m.mu.RLock()
		for _, targetID := range envelope.Targets {
			if clients, ok := m.Clients[targetID]; ok {
				payload := envelope.Payload
				// If it's a message, inject the specific user's unread count and hide others'
				if envelope.Type == "message.created" {
					if payloadMap, ok := payload.(map[string]interface{}); ok {
						// Create a copy to avoid side effects for other users in the same loop
						newPayload := make(map[string]interface{})
						for k, v := range payloadMap {
							if k != "recipient_unread_counts" {
								newPayload[k] = v
							}
						}
						// Extract this specific user's count
						if counts, ok := payloadMap["recipient_unread_counts"].(map[string]interface{}); ok {
							if count, ok := counts[targetID]; ok {
								newPayload["unread_count"] = count
							}
						}
						payload = newPayload
					}
				}

				// Send to all connections for this user
				wsPayload := CreateWSPayload(envelope.Type, payload)
				for _, client := range clients {
					select {
					case client.Send <- wsPayload:
					default:
						close(client.Send)
					}
				}
			}
		}
		m.mu.RUnlock()
	}
}

// BroadcastToRedis is called by Kafka Consumer to push logic to all Gateways
func (m *Manager) BroadcastToRedis(envelope events.RedisEnvelope) error {
	bytes, _ := json.Marshal(envelope)
	return m.Redis.Publish(context.Background(), "chat.live.updates", bytes).Err()
}

// setUserOnline updates user presence in Redis and broadcasts to all users
func (m *Manager) setUserOnline(userID string, isOnline bool) {
	ctx := context.Background()

	// Store in Redis with TTL (5 minutes for online status)
	key := "user:presence:" + userID
	if isOnline {
		m.Redis.Set(ctx, key, "online", 5*60*time.Second)
		log.Printf("User %s is now ONLINE", userID)
	} else {
		m.Redis.Del(ctx, key)
		log.Printf("User %s is now OFFLINE", userID)
	}

	// Broadcast presence change to all connected users
	presenceEvent := events.UserPresenceEvent{
		UserID:    userID,
		IsOnline:  isOnline,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	envelope := events.RedisEnvelope{
		Type:    events.TopicUserPresence,
		Payload: presenceEvent,
		Targets: []string{}, // Empty means broadcast to all
	}

	bytes, _ := json.Marshal(envelope)
	m.Redis.Publish(ctx, "chat.live.updates", bytes)
}

// IsUserOnline checks if a user is currently online
func (m *Manager) IsUserOnline(userID string) bool {
	ctx := context.Background()
	key := "user:presence:" + userID
	result, err := m.Redis.Get(ctx, key).Result()
	return err == nil && result == "online"
}

// GetOnlineUsers returns a list of currently online user IDs
func (m *Manager) GetOnlineUsers(userIDs []string) map[string]bool {
	ctx := context.Background()
	onlineStatus := make(map[string]bool)

	for _, userID := range userIDs {
		key := "user:presence:" + userID
		result, err := m.Redis.Get(ctx, key).Result()
		onlineStatus[userID] = err == nil && result == "online"
	}

	return onlineStatus
}

// KeepAlive should be called periodically to refresh user's online status
func (m *Manager) KeepAlive(userID string) {
	ctx := context.Background()
	key := "user:presence:" + userID
	m.Redis.Expire(ctx, key, 5*60*time.Second)
}
