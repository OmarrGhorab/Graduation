package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/graduation/chat-service/internal/events"
	"github.com/segmentio/kafka-go"
)

type Producer struct {
	Writer *kafka.Writer
}

func NewProducer(brokers []string) *Producer {
	return &Producer{
		Writer: &kafka.Writer{
			Addr:     kafka.TCP(brokers...),
			Balancer: &kafka.Hash{}, // Important for ordering by Key (ConversationID)
			// Async writes for performance
			Async:        true,
			BatchSize:    100,
			BatchTimeout: 10 * time.Millisecond,
		},
	}
}

func (p *Producer) Close() error {
	return p.Writer.Close()
}

// PublishMessageCreated sends a message created event
func (p *Producer) PublishMessageCreated(ctx context.Context, event events.MessageCreatedEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	return p.Writer.WriteMessages(ctx, kafka.Message{
		Topic: events.TopicMessageCreated,
		Key:   []byte(event.ConversationID), // Partition by ConversationID
		Value: payload,
	})
}

// PublishTyping sends a typing indicator event
func (p *Producer) PublishTyping(ctx context.Context, event events.TypingEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	return p.Writer.WriteMessages(ctx, kafka.Message{
		Topic: events.TopicTyping,
		Key:   []byte(event.ConversationID), // Partition by ConversationID
		Value: payload,
	})
}
