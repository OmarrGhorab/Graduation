package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/graduation/chat-service/internal/events"
	"github.com/graduation/chat-service/internal/kafka"
	"github.com/graduation/chat-service/internal/models"
	"github.com/graduation/chat-service/internal/repositories"
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
	redis    *cache.RedisClient
	producer *kafka.Producer
	repos    *repositories.Repositories
}

// NewTypingService creates a new TypingService
func NewTypingService(redis *cache.RedisClient, producer *kafka.Producer, repos *repositories.Repositories) *TypingService {
	return &TypingService{
		redis:    redis,
		producer: producer,
		repos:    repos,
	}
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

	fmt.Printf("[TypingService] SetTyping called: user=%s conversation=%s\n", userID, conversationID)

	// Get conversation members for the event
	memberIDs, err := s.getConversationMemberIDs(ctx, conversationID)
	if err != nil {
		// Log but don't fail - typing indicator is non-critical
		fmt.Printf("[TypingService] Failed to get members for typing event: %v\n", err)
	} else {
		fmt.Printf("[TypingService] Found %d members for typing event\n", len(memberIDs))
	}

	// Publish to Kafka
	if s.producer != nil {
		event := events.TypingEvent{
			UserID:         userID,
			ConversationID: conversationID,
			IsTyping:       true,
			RecipientIDs:   memberIDs, // Include recipients for routing
		}
		go func() {
			fmt.Printf("[TypingService] Publishing typing event to Kafka for %d recipients\n", len(memberIDs))
			if err := s.producer.PublishTyping(context.Background(), event); err != nil {
				fmt.Printf("[TypingService] Failed to publish event: %v\n", err)
			} else {
				fmt.Printf("[TypingService] Successfully published typing event\n")
			}
		}()
	} else {
		fmt.Printf("[TypingService] Producer is nil, skipping Kafka publish\n")
	}

	return s.redis.Set(ctx, key, value, TypingTTL)
}

// getConversationMemberIDs retrieves member IDs from cache or database
func (s *TypingService) getConversationMemberIDs(ctx context.Context, conversationID string) ([]string, error) {
	// Query Redis first for cached members
	membersKey := fmt.Sprintf("conv:members:%s", conversationID)
	members, err := s.redis.SMembers(ctx, membersKey)
	if err == nil && len(members) > 0 {
		fmt.Printf("[TypingService] Found %d cached members for conversation %s\n", len(members), conversationID)
		return members, nil
	}

	// If not in cache, fetch from database
	fmt.Printf("[TypingService] Cache miss for conversation %s, fetching from DB...\n", conversationID)
	memberIDs, err := s.repos.Member.GetConversationMemberIDs(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get members from DB: %w", err)
	}

	// Cache the results for future use
	if len(memberIDs) > 0 {
		members := make([]interface{}, len(memberIDs))
		for i, id := range memberIDs {
			members[i] = id
		}
		if err := s.redis.SAdd(ctx, membersKey, members...); err != nil {
			fmt.Printf("[TypingService] Failed to cache members: %v\n", err)
		} else {
			// Set expiration (30 days)
			s.redis.Expire(ctx, membersKey, 30*24*60*60)
			fmt.Printf("[TypingService] Cached %d members for conversation %s\n", len(memberIDs), conversationID)
		}
	}

	return memberIDs, nil
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
