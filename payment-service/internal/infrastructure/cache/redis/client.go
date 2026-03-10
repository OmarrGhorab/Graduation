package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/OmarrGhorab/payment-service/internal/config"
	"github.com/redis/go-redis/v9"
)

type Client struct {
	client *redis.Client
}

func NewClient(cfg config.RedisConfig) (*Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &Client{client: client}, nil
}

func (c *Client) Close() error {
	return c.client.Close()
}

func (c *Client) AcquireIdempotencyLock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	fullKey := fmt.Sprintf("payment:lock:webhook:%s", key)
	return c.client.SetNX(ctx, fullKey, "1", ttl).Result()
}

func (c *Client) IsTransactionProcessed(ctx context.Context, transactionID string) (bool, error) {
	key := fmt.Sprintf("payment:processed:tx:%s", transactionID)
	exists, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

func (c *Client) MarkTransactionProcessed(ctx context.Context, transactionID string, ttl time.Duration) error {
	key := fmt.Sprintf("payment:processed:tx:%s", transactionID)
	return c.client.Set(ctx, key, "1", ttl).Err()
}

func (c *Client) SetPaymentSession(ctx context.Context, orderID string, data string, ttl time.Duration) error {
	key := fmt.Sprintf("payment:session:%s", orderID)
	return c.client.Set(ctx, key, data, ttl).Err()
}

func (c *Client) GetPaymentSession(ctx context.Context, orderID string) (string, error) {
	key := fmt.Sprintf("payment:session:%s", orderID)
	return c.client.Get(ctx, key).Result()
}
