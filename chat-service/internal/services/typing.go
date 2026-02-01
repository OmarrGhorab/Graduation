package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/graduation/chat-service/internal/models"
	"github.com/graduation/chat-service/pkg/cache"
)

const (
	// TypingKeyPrefix is the Redis key prefix for typing indicators
	TypingKeyPrefix = "typing"
	// TypingTTL is how long a typing indicator lasts
	TypingTTL = 3 * time.Second
)

// TypingService handles typing indicator logic
type TypingService struct {
	redis *cache.RedisClient
}

// NewTypingService creates a new TypingService
func NewTypingService(redis *cache.RedisClient) *TypingService {
	return &TypingService{redis: redis}
}

// TypingUser represents a user who is typing
type TypingUser struct {
	UserID   string          `json:"user_id"`
	UserRole models.UserRole `json:"user_role"`
}

// SetTyping sets the typing indicator for a user in a conversation
func (s *TypingService) SetTyping(ctx context.Context, conversationID, userID string, userRole models.UserRole) error {
	key := fmt.Sprintf("%s:%s:%s", TypingKeyPrefix, conversationID, userID)
	value := string(userRole) // Store the role with the typing indicator
	return s.redis.Set(ctx, key, value, TypingTTL)
}

// GetTypingUsers gets all users currently typing in a conversation
func (s *TypingService) GetTypingUsers(ctx context.Context, conversationID string) ([]TypingUser, error) {
	pattern := fmt.Sprintf("%s:%s:*", TypingKeyPrefix, conversationID)
	keys, err := s.redis.Keys(ctx, pattern)
	if err != nil {
		return nil, err
	}

	var typingUsers []TypingUser
	for _, key := range keys {
		// Extract user ID from key: typing:conv_id:user_id
		parts := strings.Split(key, ":")
		if len(parts) != 3 {
			continue
		}
		userID := parts[2]

		// Get the role stored in the value
		roleStr, err := s.redis.Get(ctx, key)
		if err != nil {
			continue
		}

		typingUsers = append(typingUsers, TypingUser{
			UserID:   userID,
			UserRole: models.UserRole(roleStr),
		})
	}

	return typingUsers, nil
}

// ClearTyping clears the typing indicator for a user (called when message is sent)
func (s *TypingService) ClearTyping(ctx context.Context, conversationID, userID string) error {
	key := fmt.Sprintf("%s:%s:%s", TypingKeyPrefix, conversationID, userID)
	return s.redis.Del(ctx, key)
}
