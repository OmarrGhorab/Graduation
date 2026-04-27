package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

// Producer handles Kafka message publishing
type Producer struct {
	writer *kafka.Writer
}

// NewProducer creates a new Kafka producer
func NewProducer(brokers []string) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr:     kafka.TCP(brokers...),
			Balancer: &kafka.LeastBytes{},
			// Set high retries and backoff for reliability
			MaxAttempts:  5,
			BatchTimeout: 10 * time.Millisecond,
			Async:        true,
			ErrorLogger:  kafka.LoggerFunc(func(s string, i ...interface{}) { log.Printf("Kafka Error: "+s, i...) }),
		},
	}
}

// Publish serializes and sends a message to a specific topic
func (p *Producer) Publish(ctx context.Context, topic string, key string, data interface{}) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	err = p.writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: payload,
	})

	if err != nil {
		return fmt.Errorf("failed to write message to Kafka: %w", err)
	}

	return nil
}

// Close gracefully shuts down the producer
func (p *Producer) Close() error {
	return p.writer.Close()
}
