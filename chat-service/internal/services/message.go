package services

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/graduation/chat-service/internal/models"
	"github.com/graduation/chat-service/internal/repositories"
	"github.com/graduation/chat-service/pkg/cache"
)

// MessageService handles message business logic
type MessageService struct {
	repos        *repositories.Repositories
	redis        *cache.RedisClient
	notification *NotificationService
}

// NewMessageService creates a new MessageService
func NewMessageService(repos *repositories.Repositories, redis *cache.RedisClient, notification *NotificationService) *MessageService {
	return &MessageService{
		repos:        repos,
		redis:        redis,
		notification: notification,
	}
}

// SendMessageInput input for sending a message
type SendMessageInput struct {
	ConversationID string
	SenderID       string
	SenderRole     models.UserRole
	Type           models.MessageType
	Content        string
	MediaURL       string
	MediaMetadata  map[string]interface{}
	ReplyToID      *string
}

// SendMessage sends a message to a conversation
func (s *MessageService) SendMessage(ctx context.Context, input SendMessageInput) (*models.Message, error) {
	// Verify sender is a member
	isMember, err := s.repos.Member.IsMember(ctx, input.ConversationID, input.SenderID)
	if err != nil || !isMember {
		return nil, errors.New("you are not a member of this conversation")
	}

	// Create message
	message := &models.Message{
		ID:             uuid.New().String(),
		ConversationID: input.ConversationID,
		SenderID:       input.SenderID,
		SenderRole:     input.SenderRole,
		Type:           input.Type,
		Content:        input.Content,
		MediaURL:       input.MediaURL,
		ReplyToID:      input.ReplyToID,
	}

	// Handle media metadata
	if input.MediaMetadata != nil {
		metadataJSON, _ := json.Marshal(input.MediaMetadata)
		message.MediaMetadata = metadataJSON
	} else {
		message.MediaMetadata = json.RawMessage("{}")
	}

	if err := s.repos.Message.Create(ctx, message); err != nil {
		return nil, err
	}

	// Update conversation timestamp
	conv, _ := s.repos.Conversation.GetByID(ctx, input.ConversationID)
	if conv != nil {
		_ = s.repos.Conversation.Update(ctx, conv)
	}

	// Get full message with reply
	fullMessage, _ := s.repos.Message.GetByIDWithReply(ctx, message.ID)

	// Send notifications to offline members (async, non-blocking)
	go s.notifyMembers(context.Background(), input.ConversationID, input.SenderID, fullMessage)

	return fullMessage, nil
}

// notifyMembers sends push notifications to offline members
func (s *MessageService) notifyMembers(ctx context.Context, conversationID, senderID string, message *models.Message) {
	memberIDs, err := s.repos.Member.GetConversationMemberIDs(ctx, conversationID)
	if err != nil {
		return
	}

	conv, err := s.repos.Conversation.GetByID(ctx, conversationID)
	if err != nil {
		return
	}

	// Filter out sender and check who's online
	var offlineMembers []string
	for _, memberID := range memberIDs {
		if memberID == senderID {
			continue
		}
		// Check if member has active poll connection
		isPolling, _ := s.redis.Exists(ctx, "poll:"+memberID+":"+conversationID)
		if !isPolling {
			offlineMembers = append(offlineMembers, memberID)
		}
	}

	if len(offlineMembers) > 0 {
		s.notification.SendChatNotification(ctx, message, conv, offlineMembers)
	}
}

// GetMessages retrieves messages for a conversation
func (s *MessageService) GetMessages(ctx context.Context, conversationID, userID string, limit, offset int) ([]models.Message, error) {
	// Verify user is a member
	isMember, err := s.repos.Member.IsMember(ctx, conversationID, userID)
	if err != nil || !isMember {
		return nil, errors.New("you are not a member of this conversation")
	}

	return s.repos.Message.GetConversationMessages(ctx, conversationID, limit, offset)
}

// GetMessagesAfter retrieves messages after a specific message (for long polling)
func (s *MessageService) GetMessagesAfter(ctx context.Context, conversationID, afterMessageID string, limit int) ([]models.Message, error) {
	return s.repos.Message.GetMessagesAfter(ctx, conversationID, afterMessageID, limit)
}

// DeleteMessage soft deletes a message
func (s *MessageService) DeleteMessage(ctx context.Context, messageID, userID string, userRole models.UserRole) error {
	message, err := s.repos.Message.GetByID(ctx, messageID)
	if err != nil {
		return errors.New("message not found")
	}

	// Only sender or admins can delete
	if message.SenderID != userID && !canModerate(userRole) {
		return errors.New("you don't have permission to delete this message")
	}

	return s.repos.Message.SoftDelete(ctx, messageID)
}

// PinMessage pins a message
func (s *MessageService) PinMessage(ctx context.Context, conversationID, messageID, userID string, userRole models.UserRole, memberRole models.MemberRole) error {
	// Check permission
	if !canPin(userRole, memberRole) {
		return errors.New("you don't have permission to pin messages")
	}

	// Check message exists and belongs to conversation
	message, err := s.repos.Message.GetByID(ctx, messageID)
	if err != nil {
		return errors.New("message not found")
	}
	if message.ConversationID != conversationID {
		return errors.New("message does not belong to this conversation")
	}

	// Check if already pinned
	isPinned, _ := s.repos.Message.IsPinned(ctx, messageID)
	if isPinned {
		return errors.New("message is already pinned")
	}

	pinnedMsg := &models.PinnedMessage{
		ID:             uuid.New().String(),
		MessageID:      messageID,
		ConversationID: conversationID,
		PinnedBy:       userID,
	}

	return s.repos.Message.Pin(ctx, pinnedMsg)
}

// UnpinMessage unpins a message
func (s *MessageService) UnpinMessage(ctx context.Context, conversationID, messageID, userID string, userRole models.UserRole, memberRole models.MemberRole) error {
	// Check permission
	if !canPin(userRole, memberRole) {
		return errors.New("you don't have permission to unpin messages")
	}

	return s.repos.Message.Unpin(ctx, messageID)
}

// GetPinnedMessages retrieves all pinned messages for a conversation
func (s *MessageService) GetPinnedMessages(ctx context.Context, conversationID, userID string) ([]models.PinnedMessage, error) {
	// Verify user is a member
	isMember, err := s.repos.Member.IsMember(ctx, conversationID, userID)
	if err != nil || !isMember {
		return nil, errors.New("you are not a member of this conversation")
	}

	return s.repos.Message.GetPinnedMessages(ctx, conversationID)
}

// canModerate checks if a role can moderate messages
func canModerate(role models.UserRole) bool {
	return role == models.UserRoleInstructor || role == models.UserRoleTeacher || role == models.UserRoleAssistant
}

// canPin checks if a user can pin messages
func canPin(userRole models.UserRole, memberRole models.MemberRole) bool {
	// Check user role permissions
	if userRole == models.UserRoleInstructor || userRole == models.UserRoleTeacher || userRole == models.UserRoleAssistant {
		return true
	}
	// Check member role permissions
	return memberRole == models.MemberRoleOwner || memberRole == models.MemberRoleAdmin
}
