package kafka

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/graduation/ws-gateway/internal/events"
	"github.com/graduation/ws-gateway/internal/websocket"
	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	MessageReader *kafka.Reader
	TypingReader  *kafka.Reader
	Manager       *websocket.Manager
}

func NewConsumer(brokers []string, groupID string, manager *websocket.Manager) *Consumer {
	return &Consumer{
		MessageReader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:         brokers,
			GroupID:         groupID,
			Topic:           events.TopicMessageCreated,
			MinBytes:        1,                       // Read immediately
			MaxBytes:        10e6,
			MaxWait:         10 * time.Millisecond,   // Poll every 10ms
			CommitInterval:  100 * time.Millisecond,  // Commit frequently
			StartOffset:     kafka.LastOffset,        // Start from latest
			ReadBackoffMin:  10 * time.Millisecond,   // Fast retry
			ReadBackoffMax:  100 * time.Millisecond,
		}),
		TypingReader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:         brokers,
			GroupID:         groupID,
			Topic:           events.TopicTyping,
			MinBytes:        1,                       // Read immediately
			MaxBytes:        10e6,
			MaxWait:         10 * time.Millisecond,   // Poll every 10ms
			CommitInterval:  100 * time.Millisecond,  // Commit frequently
			StartOffset:     kafka.LastOffset,        // Start from latest
			ReadBackoffMin:  10 * time.Millisecond,   // Fast retry
			ReadBackoffMax:  100 * time.Millisecond,
		}),
		Manager: manager,
	}
}

func (c *Consumer) Start(ctx context.Context) {
	log.Println("Starting Kafka Consumer...")
	
	// Start message consumer
	go c.consumeMessages(ctx)
	
	// Start typing consumer
	go c.consumeTyping(ctx)
}

func (c *Consumer) consumeMessages(ctx context.Context) {
	log.Println("Starting Message Consumer...")
	for {
		m, err := c.MessageReader.ReadMessage(ctx)
		if err != nil {
			log.Printf("Kafka Read Error (messages): %v. Retrying in 1s...", err)
			time.Sleep(1 * time.Second)
			continue
		}

		var event events.MessageCreatedEvent
		if err := json.Unmarshal(m.Value, &event); err != nil {
			log.Printf("Unmarshal Error (messages): %v", err)
			continue
		}

		// Wrap and Broadcast to Redis
		envelope := events.RedisEnvelope{
			Type:    "message.created",
			Payload: event,
			Targets: event.RecipientIDs,
		}

		if err := c.Manager.BroadcastToRedis(envelope); err != nil {
			log.Printf("Redis Broadcast Error (messages): %v", err)
		} else {
			log.Printf("✅ Broadcasted message %s to %d recipients", event.ID, len(event.RecipientIDs))
		}
	}
}

func (c *Consumer) consumeTyping(ctx context.Context) {
	log.Println("Starting Typing Consumer...")
	for {
		m, err := c.TypingReader.ReadMessage(ctx)
		if err != nil {
			log.Printf("Kafka Read Error (typing): %v. Retrying in 1s...", err)
			time.Sleep(1 * time.Second)
			continue
		}

		var event events.TypingEvent
		if err := json.Unmarshal(m.Value, &event); err != nil {
			log.Printf("Unmarshal Error (typing): %v", err)
			continue
		}

		// Wrap and Broadcast to Redis
		envelope := events.RedisEnvelope{
			Type:    "typing",
			Payload: event,
			Targets: event.RecipientIDs,
		}

		if err := c.Manager.BroadcastToRedis(envelope); err != nil {
			log.Printf("Redis Broadcast Error (typing): %v", err)
		} else {
			log.Printf("✅ Broadcasted typing event from %s to %d recipients", event.UserID, len(event.RecipientIDs))
		}
	}
}
