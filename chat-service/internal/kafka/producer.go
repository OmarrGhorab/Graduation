package kafka

import (
	"context"
	"encoding/json"
	"log"

	"github.com/graduation/chat-service/internal/events"
	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(brokers []string) *Producer {
	w := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Balancer:     &kafka.Hash{}, // Ensure ordering by Key (ConversationID)
		Async:        false,          // Synchronous for immediate delivery
		BatchSize:    1,              // Send immediately, don't batch
		BatchTimeout: 1,              // 1ms timeout
		RequiredAcks: 1,              // Wait for leader acknowledgment only
		Compression:  kafka.Snappy,   // Fast compression
	}
	return &Producer{writer: w}
}

func (p *Producer) Close() {
	if err := p.writer.Close(); err != nil {
		log.Printf("Failed to close kafka writer: %v", err)
	}
}

func (p *Producer) PublishMessageCreated(event events.MessageCreatedEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	msg := kafka.Message{
		Topic: events.TopicMessageCreated,
		Key:   []byte(event.ConversationID),
		Value: payload,
	}

	return p.writer.WriteMessages(context.Background(), msg)
}

func (p *Producer) PublishTyping(event events.TypingEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	msg := kafka.Message{
		Topic: events.TopicTyping,
		Key:   []byte(event.ConversationID),
		Value: payload,
	}

	return p.writer.WriteMessages(context.Background(), msg)
}
