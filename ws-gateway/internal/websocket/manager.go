package websocket

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/redis/go-redis/v9"
)

// Client represents a connected WebSocket client
type Client struct {
	UserID   string
	UserRole string
	Conn     *websocket.Conn
	Send     chan []byte
}

// Manager maintains the set of active clients and broadcasts messages
type Manager struct {
	Clients    map[string]*Client // UserID -> Client
	Register   chan *Client
	Unregister chan *Client
	Redis      *redis.Client
	mu         sync.RWMutex
}

// NewManager creates a new key-based Manager
func NewManager(redisClient *redis.Client) *Manager {
	return &Manager{
		Clients:    make(map[string]*Client),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Redis:      redisClient,
	}
}

// Run starts the manager loop
func (m *Manager) Run() {
	for {
		select {
		case client := <-m.Register:
			m.mu.Lock()
			m.Clients[client.UserID] = client
			m.mu.Unlock()

			// Set presence in Redis
			go m.SetPresence(client.UserID, true)

			log.Printf("User registered: %s", client.UserID)

		case client := <-m.Unregister:
			m.mu.Lock()
			if _, ok := m.Clients[client.UserID]; ok {
				delete(m.Clients, client.UserID)
				close(client.Send)
			}
			m.mu.Unlock()

			// Remove presence (or let TTL expire)
			go m.SetPresence(client.UserID, false)

			log.Printf("User unregistered: %s", client.UserID)
		}
	}
}

// SetPresence updates the user's presence in Redis
func (m *Manager) SetPresence(userID string, online bool) {
	ctx := context.Background()
	key := "presence:" + userID

	if online {
		// Set online with short TTL (will be refreshed by heartbeat)
		m.Redis.Set(ctx, key, "online", 60*time.Second)
	} else {
		m.Redis.Del(ctx, key)
	}
}

// SendToUser sends a message to a specific user if connected
func (m *Manager) SendToUser(userID string, message interface{}) {
	m.mu.RLock()
	client, ok := m.Clients[userID]
	m.mu.RUnlock()

	if ok {
		payload, _ := json.Marshal(message)
		select {
		case client.Send <- payload:
		default:
			log.Printf("Failed to send to user %s, channel full", userID)
			close(client.Send)
			delete(m.Clients, userID)
		}
	}
}

// StartRedisSubscriber starts a subscriber to listen for broadcast messages from Redis
func (m *Manager) StartRedisSubscriber(ctx context.Context) {
	pubsub := m.Redis.Subscribe(ctx, "chat.live.updates")
	defer pubsub.Close()

	log.Println("[Manager] Redis subscriber started for channel: chat.live.updates")

	for {
		select {
		case <-ctx.Done():
			log.Println("[Manager] Redis subscriber shutting down")
			return
		default:
			msg, err := pubsub.ReceiveMessage(ctx)
			if err != nil {
				log.Printf("[Manager] Redis receive error: %v", err)
				continue
			}

			// Parse the routing envelope
			var envelope struct {
				RecipientIDs []string    `json:"recipient_ids"`
				Payload      interface{} `json:"payload"`
			}

			if err := json.Unmarshal([]byte(msg.Payload), &envelope); err != nil {
				log.Printf("[Manager] Failed to unmarshal envelope: %v", err)
				continue
			}

			log.Printf("[Manager] Received Redis message for %d recipients", len(envelope.RecipientIDs))

			// Send to all recipients
			sentCount := 0
			for _, userID := range envelope.RecipientIDs {
				m.SendToUser(userID, envelope.Payload)
				sentCount++
			}
			log.Printf("[Manager] Sent WebSocket message to %d/%d connected recipients", sentCount, len(envelope.RecipientIDs))
		}
	}
}
