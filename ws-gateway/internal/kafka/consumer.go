package kafka

import (
	"context"
	"encoding/json"
	"log"

	"github.com/graduation/ws-gateway/internal/events"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	Reader *kafka.Reader
	Redis  *redis.Client
}

func NewConsumer(brokers []string, groupID string, redisClient *redis.Client) *Consumer {
	return &Consumer{
		Reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:  brokers,
			GroupID:  groupID,
			Topic:    events.TopicMessageCreated,
			MinBytes: 10e3, // 10KB
			MaxBytes: 10e6, // 10MB
		}),
		Redis: redisClient,
	}
}

func (c *Consumer) Start(ctx context.Context) {
	logger := log.New(log.Writer(), "[Kafka-Consumer] ", log.LstdFlags)
	logger.Println("Starting Consumer...")

	for {
		m, err := c.Reader.ReadMessage(ctx)
		if err != nil {
			logger.Printf("Error reading message: %v", err)
			break
		}

		logger.Printf("Message received: topic=%s partition=%d offset=%d", m.Topic, m.Partition, m.Offset)

		// Unmarshal to check structure (optional validation)
		var event events.MessageCreatedEvent
		if err := json.Unmarshal(m.Value, &event); err != nil {
			logger.Printf("Error unmarshalling event: %v", err)
			continue
		}

		// FAN-OUT STRATEGY:
		// Chat Service should ideally enrich this event with "RecipientIDs".
		// Since we don't have Chat Service code updated yet, we will assume the Payload is WRAPPED
		// with a routing envelope or we fetch recipients here.
		// FETCHING RECIPIENTS HERE IS SLOW (requires DB call).
		// RECOMMENDATION: Chat Service handles the logic and puts "RecipientIDs" in the Kafka message.
		// For MVP: We will construct a RoutingEnvelope here assuming we knew the recipients.
		// Wait, if Chat Service puts RecipientIDs in the Kafka message, then we just need to forward it.

		// Construct routing envelope
		// TODO: Real implementation requires extracting RecipientIDs from the event
		// (which implies Chat Service added them).
		// For now, let's assume we broadcast to the conversation ID channel?
		// No, we decided on Redis Pub/Sub "chat.live.updates".

		// Mocking recipients for testing flow (In real code, this comes from DB/Upstream)
		recipients := []string{event.SenderID} // Echo back to sender for ack visual

		routingEnvelope := struct {
			RecipientIDs []string    `json:"recipient_ids"`
			Payload      interface{} `json:"payload"`
		}{
			RecipientIDs: recipients,
			Payload:      event,
		}

		bytes, _ := json.Marshal(routingEnvelope)

		// Publish to Redis for all Gateways to pick up
		c.Redis.Publish(ctx, "chat.live.updates", bytes)
	}
}
