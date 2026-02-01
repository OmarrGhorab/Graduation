package services

import (
	"context"
	"errors"

	"fmt"

	"github.com/google/uuid"
	"github.com/graduation/chat-service/internal/clients"
	"github.com/graduation/chat-service/internal/models"
	"github.com/graduation/chat-service/internal/repositories"
	"github.com/graduation/chat-service/pkg/cache"
)

// ConversationService handles conversation business logic
type ConversationService struct {
	repos        *repositories.Repositories
	redis        *cache.RedisClient
	notification *NotificationService
	authClient   *clients.AuthClient
}

// NewConversationService creates a new ConversationService
func NewConversationService(repos *repositories.Repositories, redis *cache.RedisClient, notification *NotificationService, authClient *clients.AuthClient) *ConversationService {
	return &ConversationService{
		repos:        repos,
		redis:        redis,
		notification: notification,
		authClient:   authClient,
	}
}

// CreateGroupInput input for creating a group chat
type CreateGroupInput struct {
	Name        string
	Description string
	MemberIDs   []string
	CreatorID   string
	CreatorRole models.UserRole
}

// CreateGroup creates a new group conversation
func (s *ConversationService) CreateGroup(ctx context.Context, input CreateGroupInput) (*models.Conversation, error) {
	// Check permission
	if !canCreateGroup(input.CreatorRole) {
		return nil, errors.New("only instructors, teachers, and assistants can create group chats")
	}

	// Create conversation
	conversation := &models.Conversation{
		ID:          uuid.New().String(),
		Type:        models.ConversationTypeGroup,
		Name:        input.Name,
		Description: input.Description,
		CreatedBy:   input.CreatorID,
	}

	if err := s.repos.Conversation.Create(ctx, conversation); err != nil {
		return nil, err
	}

	// Add creator as owner
	creatorMember := &models.ConversationMember{
		ID:             uuid.New().String(),
		ConversationID: conversation.ID,
		UserID:         input.CreatorID,
		UserRole:       input.CreatorRole,
		MemberRole:     models.MemberRoleOwner,
	}
	if err := s.repos.Member.Create(ctx, creatorMember); err != nil {
		return nil, err
	}

	// Add other members
	for _, memberID := range input.MemberIDs {
		if memberID == input.CreatorID {
			continue // Skip creator, already added
		}
		member := &models.ConversationMember{
			ID:             uuid.New().String(),
			ConversationID: conversation.ID,
			UserID:         memberID,
			UserRole:       models.UserRoleStudent, // Default role, can be updated later
			MemberRole:     models.MemberRoleMember,
		}
		if err := s.repos.Member.Create(ctx, member); err != nil {
			return nil, err
		}
	}

	conv, err := s.repos.Conversation.GetByIDWithMembers(ctx, conversation.ID)
	if err != nil {
		return nil, err
	}
	if err := s.enrichMembers(ctx, conv); err != nil {
		// Log error but return conversation
		fmt.Printf("[ConversationService] Failed to enrich members: %v\n", err)
	}
	return conv, nil
}

// CreateDirectChat creates or retrieves a direct chat between two users
func (s *ConversationService) CreateDirectChat(ctx context.Context, userID, recipientID string, userRole models.UserRole) (*models.Conversation, error) {
	// Check if direct chat already exists
	existing, err := s.repos.Conversation.FindDirectChat(ctx, userID, recipientID)
	if err == nil {
		return s.repos.Conversation.GetByIDWithMembers(ctx, existing.ID)
	}

	// Create new direct chat
	conversation := &models.Conversation{
		ID:        uuid.New().String(),
		Type:      models.ConversationTypeDirect,
		CreatedBy: userID,
	}

	if err := s.repos.Conversation.Create(ctx, conversation); err != nil {
		return nil, err
	}

	// Add both users as members
	for _, memberData := range []struct {
		userID string
		role   models.UserRole
	}{
		{userID, userRole},
		{recipientID, models.UserRoleStudent}, // Default, can be updated
	} {
		member := &models.ConversationMember{
			ID:             uuid.New().String(),
			ConversationID: conversation.ID,
			UserID:         memberData.userID,
			UserRole:       memberData.role,
			MemberRole:     models.MemberRoleMember,
		}
		if err := s.repos.Member.Create(ctx, member); err != nil {
			return nil, err
		}
	}

	conv, err := s.repos.Conversation.GetByIDWithMembers(ctx, conversation.ID)
	if err != nil {
		return nil, err
	}
	if err := s.enrichMembers(ctx, conv); err != nil {
		fmt.Printf("[ConversationService] Failed to enrich members: %v\n", err)
	}
	return conv, nil
}

// GetByID retrieves a conversation by ID
func (s *ConversationService) GetByID(ctx context.Context, id string) (*models.Conversation, error) {
	conv, err := s.repos.Conversation.GetByIDWithMembers(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.enrichMembers(ctx, conv); err != nil {
		fmt.Printf("[ConversationService] Failed to enrich members: %v\n", err)
	}
	return conv, nil
}

// GetMembers retrieves all members for a conversation
func (s *ConversationService) GetMembers(ctx context.Context, conversationID, userID string) ([]models.ConversationMember, error) {
	// Verify membership
	isMember, err := s.repos.Member.IsMember(ctx, conversationID, userID)
	if err != nil || !isMember {
		return nil, errors.New("you are not a member of this conversation")
	}

	conv, err := s.repos.Conversation.GetByIDWithMembers(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if err := s.enrichMembers(ctx, conv); err != nil {
		fmt.Printf("[ConversationService] Failed to enrich members: %v\n", err)
	}
	return conv.Members, nil
}

// MarkAsRead marks all messages in a conversation as read for a user
// MarkAsRead marks all messages in a conversation as read for a user
func (s *ConversationService) MarkAsRead(ctx context.Context, conversationID, userID string, messageID *string) error {
	// Verify membership
	isMember, err := s.repos.Member.IsMember(ctx, conversationID, userID)
	if err != nil || !isMember {
		return errors.New("you are not a member of this conversation")
	}

	return s.repos.Member.ResetUnreadCount(ctx, conversationID, userID, messageID)
}

// GetUserConversations retrieves all conversations for a user
func (s *ConversationService) GetUserConversations(ctx context.Context, userID string, filter repositories.ConversationFilter, limit, offset int) ([]models.Conversation, error) {
	conversations, err := s.repos.Conversation.GetUserConversations(ctx, userID, filter, limit, offset)
	if err != nil {
		return nil, err
	}

	// Collected user IDs for enrichment
	senderIDs := make(map[string]bool)
	for _, conv := range conversations {
		if conv.LastMessageSenderID != "" {
			senderIDs[conv.LastMessageSenderID] = true
		}
	}

	// Fetch user details if needed
	if len(senderIDs) > 0 {
		ids := make([]string, 0, len(senderIDs))
		for id := range senderIDs {
			ids = append(ids, id)
		}

		users, err := s.authClient.GetBatchUsers(ctx, ids)
		if err == nil {
			// Map names to conversations
			for i := range conversations {
				senderID := conversations[i].LastMessageSenderID
				if senderID == "" {
					continue
				}

				senderName := "Unknown"
				if user, ok := users[senderID]; ok {
					senderName = user.Name
				}

				// Format: "John Doe: Hello world"
				// You can simply check if LastMessageContent is not empty
				if conversations[i].LastMessageContent != "" {
					conversations[i].PreviewText = senderName + ": " + conversations[i].LastMessageContent
				} else {
					// Handle cases like images or files where content might be empty
					// Since we didn't fetch Type, we'll genericize it or leave empty
					conversations[i].PreviewText = senderName + ": sent a message"
				}
			}
		}
	}

	return conversations, nil
}

// AddMember adds a member to a group conversation
func (s *ConversationService) AddMember(ctx context.Context, conversationID, requesterID, newMemberID string, memberRole models.MemberRole) error {
	// Check requester is owner
	requester, err := s.repos.Member.GetByConversationAndUser(ctx, conversationID, requesterID)
	if err != nil {
		return errors.New("you are not a member of this conversation")
	}
	if requester.MemberRole != models.MemberRoleOwner && requester.MemberRole != models.MemberRoleAdmin {
		return errors.New("only owner or admin can add members")
	}

	// Add new member
	member := &models.ConversationMember{
		ID:             uuid.New().String(),
		ConversationID: conversationID,
		UserID:         newMemberID,
		UserRole:       models.UserRoleStudent,
		MemberRole:     memberRole,
	}
	return s.repos.Member.Create(ctx, member)
}

// RemoveMember removes a member from a group conversation
func (s *ConversationService) RemoveMember(ctx context.Context, conversationID, requesterID, memberID string) error {
	// Check if user is leaving (removing themselves)
	if requesterID == memberID {
		return s.repos.Member.Remove(ctx, conversationID, memberID)
	}

	// Check requester is owner/admin for removing others
	requester, err := s.repos.Member.GetByConversationAndUser(ctx, conversationID, requesterID)
	if err != nil {
		return errors.New("you are not a member of this conversation")
	}
	if requester.MemberRole != models.MemberRoleOwner && requester.MemberRole != models.MemberRoleAdmin {
		return errors.New("only owner or admin can remove members")
	}

	return s.repos.Member.Remove(ctx, conversationID, memberID)
}

// UpdateMemberRole updates a member's role
func (s *ConversationService) UpdateMemberRole(ctx context.Context, conversationID, requesterID, memberID string, newRole models.MemberRole) error {
	// Check requester is owner
	requester, err := s.repos.Member.GetByConversationAndUser(ctx, conversationID, requesterID)
	if err != nil {
		return errors.New("you are not a member of this conversation")
	}
	if requester.MemberRole != models.MemberRoleOwner {
		return errors.New("only owner can update member roles")
	}

	return s.repos.Member.UpdateRole(ctx, conversationID, memberID, newRole)
}

// IsMember checks if a user is a member of a conversation
func (s *ConversationService) IsMember(ctx context.Context, conversationID, userID string) (bool, error) {
	return s.repos.Member.IsMember(ctx, conversationID, userID)
}

// GetMember retrieves a member from a conversation
func (s *ConversationService) GetMember(ctx context.Context, conversationID, userID string) (*models.ConversationMember, error) {
	return s.repos.Member.GetByConversationAndUser(ctx, conversationID, userID)
}

// canCreateGroup checks if a user role can create group chats
func canCreateGroup(role models.UserRole) bool {
	return role == models.UserRoleInstructor || role == models.UserRoleTeacher || role == models.UserRoleAssistant
}

func (s *ConversationService) enrichMembers(ctx context.Context, conv *models.Conversation) error {
	if conv == nil || len(conv.Members) == 0 {
		return nil
	}

	userIDs := make([]string, 0, len(conv.Members))
	for _, m := range conv.Members {
		userIDs = append(userIDs, m.UserID)
	}

	users, err := s.authClient.GetBatchUsers(ctx, userIDs)
	if err != nil {
		return err
	}

	for i := range conv.Members {
		if u, ok := users[conv.Members[i].UserID]; ok {
			conv.Members[i].UserName = u.Name
			conv.Members[i].UserImage = u.Image
		}
	}

	return nil
}
