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
	ReadReader    *kafka.Reader
	Manager       *websocket.Manager
}

func NewConsumer(brokers []string, groupID string, manager *websocket.Manager) *Consumer {
	readerConfig := func(topic string) kafka.ReaderConfig {
		return kafka.ReaderConfig{
			Brokers:        brokers,
			GroupID:        groupID,
			Topic:          topic,
			MinBytes:       1,
			MaxBytes:       10e6,
			MaxWait:        10 * time.Millisecond,
			CommitInterval: 100 * time.Millisecond,
			StartOffset:    kafka.LastOffset,
			ReadBackoffMin: 10 * time.Millisecond,
			ReadBackoffMax: 100 * time.Millisecond,
		}
	}

	return &Consumer{
		MessageReader: kafka.NewReader(readerConfig(events.TopicMessageCreated)),
		TypingReader:  kafka.NewReader(readerConfig(events.TopicTyping)),
		ReadReader:    kafka.NewReader(readerConfig(events.TopicConversationRead)),
		Manager:       manager,
	}
}

func (c *Consumer) Start(ctx context.Context) {
	log.Println("Starting Kafka Consumer...")

	// Start message consumer
	go c.consumeMessages(ctx)

	// Start typing consumer
	go c.consumeTyping(ctx)

	// Start read status consumer
	go c.consumeRead(ctx)
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

func (c *Consumer) consumeRead(ctx context.Context) {
	log.Println("Starting Read Status Consumer...")
	for {
		m, err := c.ReadReader.ReadMessage(ctx)
		if err != nil {
			log.Printf("Kafka Read Error (read): %v. Retrying in 1s...", err)
			time.Sleep(1 * time.Second)
			continue
		}

		var event events.ConversationReadEvent
		if err := json.Unmarshal(m.Value, &event); err != nil {
			log.Printf("Unmarshal Error (read): %v", err)
			continue
		}

		// Update event to clear unread count for the user
		envelope := events.RedisEnvelope{
			Type:    "conversation.read",
			Payload: event,
			Targets: []string{event.UserID}, // Send ONLY to the user who read it (to sync their tabs)
		}

		if err := c.Manager.BroadcastToRedis(envelope); err != nil {
			log.Printf("Redis Broadcast Error (read): %v", err)
		}
	}
}
