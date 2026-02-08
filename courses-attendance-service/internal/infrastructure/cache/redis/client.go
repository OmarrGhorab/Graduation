package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/OmarrGhorab/courses-attendance-service/internal/config"
	"github.com/redis/go-redis/v9"
)

// Client wraps the Redis client
type Client struct {
	client *redis.Client
}

// NewClient creates a new Redis client
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

// Close closes the Redis connection
func (c *Client) Close() error {
	return c.client.Close()
}

// ========== QR Token Operations ==========

// QRTokenKey returns the key for the active QR token
func QRTokenKey(lessonID string) string {
	return fmt.Sprintf("attendance:lesson:%s:active_qr", lessonID)
}

// QRNonceKey returns the key for a QR nonce
func QRNonceKey(lessonID, nonce string) string {
	return fmt.Sprintf("attendance:lesson:%s:nonce:%s", lessonID, nonce)
}

// QRTokenData represents the QR token stored in Redis
type QRTokenData struct {
	LessonID  string    `json:"lessonId"`
	Nonce     string    `json:"nonce"`
	Payload   string    `json:"payload"`
	Signature string    `json:"signature"`
	IssuedAt  time.Time `json:"issuedAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// SetActiveQRToken sets the active QR token for a lesson
func (c *Client) SetActiveQRToken(ctx context.Context, lessonID string, token QRTokenData, ttl time.Duration) error {
	data, err := json.Marshal(token)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, QRTokenKey(lessonID), data, ttl).Err()
}

// GetActiveQRToken gets the active QR token for a lesson
func (c *Client) GetActiveQRToken(ctx context.Context, lessonID string) (*QRTokenData, error) {
	data, err := c.client.Get(ctx, QRTokenKey(lessonID)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var token QRTokenData
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, err
	}
	return &token, nil
}

// DeleteActiveQRToken deletes the active QR token
func (c *Client) DeleteActiveQRToken(ctx context.Context, lessonID string) error {
	return c.client.Del(ctx, QRTokenKey(lessonID)).Err()
}

// SetQRNonce marks a nonce as used (exists until expiry)
func (c *Client) SetQRNonce(ctx context.Context, lessonID, nonce string, ttl time.Duration) error {
	return c.client.Set(ctx, QRNonceKey(lessonID, nonce), "1", ttl).Err()
}

// CheckQRNonce checks if a nonce exists (not consumed)
func (c *Client) CheckQRNonce(ctx context.Context, lessonID, nonce string) (bool, error) {
	exists, err := c.client.Exists(ctx, QRNonceKey(lessonID, nonce)).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

// ConsumeQRNonce marks a nonce as consumed by deleting it
func (c *Client) ConsumeQRNonce(ctx context.Context, lessonID, nonce string) error {
	return c.client.Del(ctx, QRNonceKey(lessonID, nonce)).Err()
}

// ========== Scan Lock Operations ==========

// ScanLockKey returns the key for a scan lock
func ScanLockKey(lessonID, studentID string) string {
	return fmt.Sprintf("attendance:lock:scan:%s:%s", lessonID, studentID)
}

// AcquireScanLock acquires a lock for a student scanning attendance
func (c *Client) AcquireScanLock(ctx context.Context, lessonID, studentID string, ttl time.Duration) (bool, error) {
	result, err := c.client.SetNX(ctx, ScanLockKey(lessonID, studentID), "1", ttl).Result()
	if err != nil {
		return false, err
	}
	return result, nil
}

// ReleaseScanLock releases a scan lock
func (c *Client) ReleaseScanLock(ctx context.Context, lessonID, studentID string) error {
	return c.client.Del(ctx, ScanLockKey(lessonID, studentID)).Err()
}

// ========== Rate Limiting ==========

// RateLimitKey returns the key for rate limiting
func RateLimitKey(userID string) string {
	return fmt.Sprintf("attendance:ratelimit:scan:%s", userID)
}

// CheckRateLimit checks if a user has exceeded the rate limit
func (c *Client) CheckRateLimit(ctx context.Context, userID string, maxAttempts int, window time.Duration) (bool, error) {
	key := RateLimitKey(userID)

	count, err := c.client.Incr(ctx, key).Result()
	if err != nil {
		return false, err
	}

	// Set expiry on first increment
	if count == 1 {
		c.client.Expire(ctx, key, window)
	}

	return count <= int64(maxAttempts), nil
}

// ========== Session Presence ==========

// PresenceKey returns the key for user presence in a lesson
func PresenceKey(lessonID, userID string) string {
	return fmt.Sprintf("attendance:presence:lesson:%s:user:%s", lessonID, userID)
}

// SetPresence marks a user as present in a lesson
func (c *Client) SetPresence(ctx context.Context, lessonID, userID string, ttl time.Duration) error {
	return c.client.Set(ctx, PresenceKey(lessonID, userID), "1", ttl).Err()
}

// GetPresenceCount returns the number of present users in a lesson
func (c *Client) GetPresenceCount(ctx context.Context, lessonID string) (int64, error) {
	pattern := fmt.Sprintf("attendance:presence:lesson:%s:user:*", lessonID)
	keys, err := c.client.Keys(ctx, pattern).Result()
	if err != nil {
		return 0, err
	}
	return int64(len(keys)), nil
}
