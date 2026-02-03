package websocket

import (
	"context"
	"encoding/json"
	"log"
	"sync"

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
			m.Clients[client.UserID] = append(m.Clients[client.UserID], client)
			m.mu.Unlock()
			log.Printf("User registered: %s", client.UserID)

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
				if len(m.Clients[client.UserID]) == 0 {
					delete(m.Clients, client.UserID)
				}
			}
			m.mu.Unlock()
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

		// Find local Targets
		m.mu.RLock()
		for _, targetID := range envelope.Targets {
			if clients, ok := m.Clients[targetID]; ok {
				// Send to all connections for this user
				wsPayload := CreateWSPayload(envelope.Type, envelope.Payload)
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
