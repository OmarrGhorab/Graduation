package services

import (
	"context"
	"fmt"
	"time"

	"github.com/graduation/chat-service/internal/models"
	"github.com/graduation/chat-service/internal/repositories"
	"github.com/graduation/chat-service/pkg/cache"
)

const (
	// PollKeyPrefix is the Redis key prefix for active poll connections
	PollKeyPrefix = "poll"
)

// PollService handles long polling logic
type PollService struct {
	messageRepo  *repositories.MessageRepository
	redis        *cache.RedisClient
	pollTimeout  time.Duration
	pollInterval time.Duration
}

// NewPollService creates a new PollService
func NewPollService(messageRepo *repositories.MessageRepository, redis *cache.RedisClient, timeout, interval time.Duration) *PollService {
	return &PollService{
		messageRepo:  messageRepo,
		redis:        redis,
		pollTimeout:  timeout,
		pollInterval: interval,
	}
}

// PollResponse is the response from a poll request
type PollResponse struct {
	Messages []models.Message `json:"messages"`
	HasMore  bool             `json:"has_more"`
}

// PollMessages polls for new messages in a conversation
func (s *PollService) PollMessages(ctx context.Context, userID, conversationID, afterMessageID string) (*PollResponse, error) {
	// Set poll connection in Redis (marks user as online for this conversation)
	pollKey := fmt.Sprintf("%s:%s:%s", PollKeyPrefix, userID, conversationID)
	pollTTL := s.pollTimeout + (5 * time.Second) // Slightly longer than timeout
	s.redis.Set(ctx, pollKey, "1", pollTTL)
	defer s.redis.Del(ctx, pollKey)

	deadline := time.Now().Add(s.pollTimeout)
	ticker := time.NewTicker(s.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			// Check for new messages
			messages, err := s.messageRepo.GetMessagesAfter(ctx, conversationID, afterMessageID, 50)
			if err != nil {
				return nil, err
			}

			if len(messages) > 0 {
				return &PollResponse{
					Messages: messages,
					HasMore:  len(messages) >= 50,
				}, nil
			}

			// Check timeout
			if time.Now().After(deadline) {
				return &PollResponse{
					Messages: []models.Message{},
					HasMore:  false,
				}, nil
			}

			// Refresh poll TTL
			s.redis.Expire(ctx, pollKey, pollTTL)
		}
	}
}

// IsUserPolling checks if a user has an active poll connection for a conversation
func (s *PollService) IsUserPolling(ctx context.Context, userID, conversationID string) (bool, error) {
	key := fmt.Sprintf("%s:%s:%s", PollKeyPrefix, userID, conversationID)
	return s.redis.Exists(ctx, key)
}
