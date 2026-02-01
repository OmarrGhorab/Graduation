package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/graduation/chat-service/internal/clients"
	"github.com/graduation/chat-service/internal/models"
	"github.com/graduation/chat-service/internal/repositories"
	"github.com/graduation/chat-service/pkg/cache"
)

// MessageService handles message business logic
type MessageService struct {
	repos        *repositories.Repositories
	redis        *cache.RedisClient
	notification *NotificationService
	authClient   *clients.AuthClient
}

// NewMessageService creates a new MessageService
func NewMessageService(repos *repositories.Repositories, redis *cache.RedisClient, notification *NotificationService, authClient *clients.AuthClient) *MessageService {
	return &MessageService{
		repos:        repos,
		redis:        redis,
		notification: notification,
		authClient:   authClient,
	}
}

// SendMessageInput input for sending a message
type SendMessageInput struct {
	ConversationID string
	SenderID       string
	SenderRole     models.UserRole
	Type           models.MessageType
	Content        string
	MediaURLs      []string
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
		MediaURLs:      input.MediaURLs,
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

	// Enrich with sender details (single user)
	users, _ := s.authClient.GetBatchUsers(ctx, []string{input.SenderID})
	if user, ok := users[input.SenderID]; ok {
		fullMessage.SenderName = user.Name
		fullMessage.SenderImage = user.Image
	}

	// Send notifications to offline members (async, non-blocking)
	go s.notifyMembers(context.Background(), input.ConversationID, input.SenderID, fullMessage)

	// Increment unread count for other members
	go func() {
		_ = s.repos.Member.IncrementUnreadCount(context.Background(), input.ConversationID, []string{input.SenderID})
	}()

	return fullMessage, nil
}

// notifyMembers sends push notifications to offline members
func (s *MessageService) notifyMembers(ctx context.Context, conversationID, senderID string, message *models.Message) {
	fmt.Printf("[DEBUG] NotifyMembers started for Conv: %s, Sender: %s\n", conversationID, senderID)

	memberIDs, err := s.repos.Member.GetConversationMemberIDs(ctx, conversationID)
	if err != nil {
		fmt.Printf("[DEBUG] Failed to get members: %v\n", err)
		return
	}
	fmt.Printf("[DEBUG] Found %d members\n", len(memberIDs))

	conv, err := s.repos.Conversation.GetByID(ctx, conversationID)
	if err != nil {
		fmt.Printf("[DEBUG] Failed to get conversation: %v\n", err)
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
		fmt.Printf("[DEBUG] Member %s isPolling: %v\n", memberID, isPolling)

		// FIXME: For debugging, we can temporarily disable this check or invert it if needed.
		// For now, let's keep it but logging is key.
		if !isPolling {
			offlineMembers = append(offlineMembers, memberID)
		}
	}

	fmt.Printf("[DEBUG] Offline members to notify: %d\n", len(offlineMembers))

	if len(offlineMembers) > 0 {
		err := s.notification.SendChatNotification(ctx, message, conv, offlineMembers)
		if err != nil {
			fmt.Printf("[DEBUG] SendChatNotification error: %v\n", err)
		} else {
			fmt.Printf("[DEBUG] SendChatNotification success\n")
		}
	}
}

// GetMessages retrieves messages for a conversation
func (s *MessageService) GetMessages(ctx context.Context, conversationID, userID string, query string, limit, offset int) ([]models.Message, error) {
	// Verify user is a member
	isMember, err := s.repos.Member.IsMember(ctx, conversationID, userID)
	if err != nil || !isMember {
		return nil, errors.New("you are not a member of this conversation")
	}

	messages, err := s.repos.Message.GetConversationMessages(ctx, conversationID, query, limit, offset)
	if err != nil {
		return nil, err
	}

	return s.enrichMessages(ctx, messages)
}

// GetMessagesAfter retrieves messages after a specific message (for long polling)
func (s *MessageService) GetMessagesAfter(ctx context.Context, conversationID, afterMessageID string, limit int) ([]models.Message, error) {
	messages, err := s.repos.Message.GetMessagesAfter(ctx, conversationID, afterMessageID, limit)
	if err != nil {
		return nil, err
	}

	return s.enrichMessages(ctx, messages)
}

// enrichMessages adds user details to messages
func (s *MessageService) enrichMessages(ctx context.Context, messages []models.Message) ([]models.Message, error) {
	if len(messages) == 0 {
		return messages, nil
	}

	// Collect unique sender IDs
	senderIDs := make(map[string]bool)
	for _, msg := range messages {
		senderIDs[msg.SenderID] = true
	}

	ids := make([]string, 0, len(senderIDs))
	for id := range senderIDs {
		ids = append(ids, id)
	}

	// Fetch user details
	fmt.Printf("[MessageService] Fetching details for %d users: %v\n", len(ids), ids)
	users, err := s.authClient.GetBatchUsers(ctx, ids)
	if err != nil {
		fmt.Printf("[MessageService] Failed to enrich: %v\n", err)
		return messages, nil
	}

	// Map details to messages
	matches := 0
	for i := range messages {
		if user, ok := users[messages[i].SenderID]; ok {
			messages[i].SenderName = user.Name
			messages[i].SenderImage = user.Image
			matches++
		}
	}
	fmt.Printf("[MessageService] Enriched %d messages\n", matches)

	return messages, nil
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

// EditMessage edits a message content
func (s *MessageService) EditMessage(ctx context.Context, messageID, userID, content string) (*models.Message, error) {
	message, err := s.repos.Message.GetByID(ctx, messageID)
	if err != nil {
		return nil, errors.New("message not found")
	}

	// Only sender can edit
	if message.SenderID != userID {
		return nil, errors.New("you can only edit your own messages")
	}

	// Cannot edit deleted messages
	if message.IsDeleted {
		return nil, errors.New("cannot edit deleted message")
	}

	// Update content
	if err := s.repos.Message.UpdateContent(ctx, messageID, content); err != nil {
		return nil, err
	}

	return s.repos.Message.GetByIDWithReply(ctx, messageID)
}

// PinMessage pins a message
func (s *MessageService) PinMessage(ctx context.Context, conversationID, messageID, userID string, userRole models.UserRole) error {
	// Get member role
	member, err := s.repos.Member.GetByConversationAndUser(ctx, conversationID, userID)
	if err != nil {
		return errors.New("you are not a member of this conversation")
	}

	// Check permission
	if !canPin(userRole, member.MemberRole) {
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

	// Clear any existing pins in this conversation (ensure one pin only)
	if err := s.repos.Message.UnpinAllInConversation(ctx, conversationID); err != nil {
		return err
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
func (s *MessageService) UnpinMessage(ctx context.Context, conversationID, messageID, userID string, userRole models.UserRole) error {
	// Get member role
	member, err := s.repos.Member.GetByConversationAndUser(ctx, conversationID, userID)
	if err != nil {
		return errors.New("you are not a member of this conversation")
	}

	// Check permission
	if !canPin(userRole, member.MemberRole) {
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

	pinnedMsgs, err := s.repos.Message.GetPinnedMessages(ctx, conversationID)
	if err != nil || len(pinnedMsgs) == 0 {
		return pinnedMsgs, err
	}

	// Extract messages for enrichment
	msgsToEnrich := make([]models.Message, 0, len(pinnedMsgs))
	for _, pm := range pinnedMsgs {
		if pm.Message != nil {
			msgsToEnrich = append(msgsToEnrich, *pm.Message)
		}
	}

	if len(msgsToEnrich) > 0 {
		enriched, err := s.enrichMessages(ctx, msgsToEnrich)
		if err == nil {
			// Map enriched messages back to pinned messages
			msgMap := make(map[string]models.Message)
			for _, m := range enriched {
				msgMap[m.ID] = m
			}

			for i := range pinnedMsgs {
				if pinnedMsgs[i].Message != nil {
					if enrichedMsg, ok := msgMap[pinnedMsgs[i].Message.ID]; ok {
						pinnedMsgs[i].Message = &enrichedMsg
					}
				}
			}
		}
	}

	return pinnedMsgs, nil
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
