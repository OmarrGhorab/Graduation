package kafka

import (
	"context"
	"encoding/json"
	"log"

	"github.com/graduation/ws-gateway/internal/events"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
)

// TypingConsumer handles typing indicator events from Kafka
type TypingConsumer struct {
	Reader *kafka.Reader
	Redis  *redis.Client
}

// NewTypingConsumer creates a new consumer for typing events
func NewTypingConsumer(brokers []string, groupID string, redisClient *redis.Client) *TypingConsumer {
	return &TypingConsumer{
		Reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:  brokers,
			GroupID:  groupID + "-typing", // Separate consumer group
			Topic:    events.TopicTyping,
			MinBytes: 10e3, // 10KB
			MaxBytes: 10e6, // 10MB
		}),
		Redis: redisClient,
	}
}

// Start begins consuming typing events
func (c *TypingConsumer) Start(ctx context.Context) {
	logger := log.New(log.Writer(), "[Kafka-TypingConsumer] ", log.LstdFlags)
	logger.Println("Starting Typing Consumer...")

	for {
		m, err := c.Reader.ReadMessage(ctx)
		if err != nil {
			logger.Printf("Error reading typing message: %v", err)
			break
		}

		logger.Printf("Typing event received: topic=%s partition=%d offset=%d", m.Topic, m.Partition, m.Offset)

		// Unmarshal typing event
		var event events.TypingEvent
		if err := json.Unmarshal(m.Value, &event); err != nil {
			logger.Printf("Error unmarshalling typing event: %v", err)
			continue
		}

		logger.Printf("Typing event: user=%s conversation=%s is_typing=%v recipients=%d",
			event.UserID, event.ConversationID, event.IsTyping, len(event.RecipientIDs))

		// Filter out the sender from recipients (don't send typing indicator back to sender)
		recipients := make([]string, 0, len(event.RecipientIDs))
		for _, recipientID := range event.RecipientIDs {
			if recipientID != event.UserID {
				recipients = append(recipients, recipientID)
			}
		}

		if len(recipients) == 0 {
			logger.Printf("No recipients for typing event (conversation=%s, user=%s)", event.ConversationID, event.UserID)
			continue
		}

		// Create enriched payload for clients
		typingPayload := struct {
			UserID         string `json:"user_id"`
			ConversationID string `json:"conversation_id"`
			IsTyping       bool   `json:"is_typing"`
			EventType      string `json:"event_type"`
		}{
			UserID:         event.UserID,
			ConversationID: event.ConversationID,
			IsTyping:       event.IsTyping,
			EventType:      "typing.user",
		}

		routingEnvelope := struct {
			RecipientIDs []string    `json:"recipient_ids"`
			Payload      interface{} `json:"payload"`
		}{
			RecipientIDs: recipients,
			Payload:      typingPayload,
		}

		bytes, _ := json.Marshal(routingEnvelope)

		// Publish to Redis for all Gateways to pick up
		if err := c.Redis.Publish(ctx, "chat.live.updates", bytes).Err(); err != nil {
			logger.Printf("Failed to publish to Redis: %v", err)
			continue
		}
		logger.Printf("Broadcasted typing event to %d recipients", len(recipients))
	}
}
