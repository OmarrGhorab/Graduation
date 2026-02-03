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

// ConversationService handles conversation business logic
type ConversationService struct {
	repos        *repositories.Repositories
	redis        *cache.RedisClient
	notification *NotificationService
	authClient   *clients.AuthClient
	media        *MediaService
}

// NewConversationService creates a new ConversationService
func NewConversationService(repos *repositories.Repositories, redis *cache.RedisClient, notification *NotificationService, authClient *clients.AuthClient, media *MediaService) *ConversationService {
	return &ConversationService{
		repos:        repos,
		redis:        redis,
		notification: notification,
		authClient:   authClient,
		media:        media,
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

	// Collect all member IDs for caching
	allMemberIDs := []string{input.CreatorID}

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
		allMemberIDs = append(allMemberIDs, memberID)
	}

	// Cache members in Redis for quick access by typing service
	s.cacheConversationMembers(ctx, conversation.ID, allMemberIDs)

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
	memberIDs := []string{userID, recipientID}
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

	// Cache members in Redis
	s.cacheConversationMembers(ctx, conversation.ID, memberIDs)

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
	
	// Cache members in Redis for typing indicator service
	if len(conv.Members) > 0 {
		memberIDs := make([]string, 0, len(conv.Members))
		for _, m := range conv.Members {
			memberIDs = append(memberIDs, m.UserID)
		}
		s.cacheConversationMembers(ctx, conv.ID, memberIDs)
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
	userIDsToFetch := make(map[string]bool)
	systemID := "00000000-0000-0000-0000-000000000000"
	for _, conv := range conversations {
		if conv.LastMessageSenderID != "" && conv.LastMessageSenderID != systemID {
			userIDsToFetch[conv.LastMessageSenderID] = true
		}
		// Collect IDs from ALL members to enrich their details (name, image)
		for _, member := range conv.Members {
			if member.UserID != systemID {
				userIDsToFetch[member.UserID] = true
			}
		}
	}

	// Fetch user details if needed
	if len(userIDsToFetch) > 0 {
		ids := make([]string, 0, len(userIDsToFetch))
		for id := range userIDsToFetch {
			ids = append(ids, id)
		}

		users, err := s.authClient.GetBatchUsers(ctx, ids)
		if err == nil {
			// Map names to conversations
			for i := range conversations {
				// Enrich Last Message Preview
				senderID := conversations[i].LastMessageSenderID
				senderName := "Unknown"
				if senderID == "00000000-0000-0000-0000-000000000000" {
					senderName = "System"
				} else if senderID != "" {
					if user, ok := users[senderID]; ok {
						senderName = user.Name
					}
				}

				// Format preview text
				if conversations[i].LastMessageContent != "" {
					if senderID == "00000000-0000-0000-0000-000000000000" {
						conversations[i].PreviewText = conversations[i].LastMessageContent
					} else {
						conversations[i].PreviewText = senderName + ": " + conversations[i].LastMessageContent
					}
				} else if senderID != "" {
					conversations[i].PreviewText = senderName + ": sent a message"
				} else {
					conversations[i].PreviewText = ""
				}

				// Enrich Member Details (for all members)
				for j := range conversations[i].Members {
					userID := conversations[i].Members[j].UserID
					if u, ok := users[userID]; ok {
						conversations[i].Members[j].UserName = u.Name
						conversations[i].Members[j].UserImage = u.Image
					}
				}

				// Enrich Direct Chat Details (Name & Image) using the OTHER member
				if conversations[i].Type == models.ConversationTypeDirect {
					for _, member := range conversations[i].Members {
						if member.UserID != userID {
							if u, ok := users[member.UserID]; ok {
								conversations[i].Name = u.Name
								conversations[i].ImageURL = u.Image
							}
							break
						}
					}
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
func (s *ConversationService) RemoveMember(ctx context.Context, conversationID, requesterID string, requesterRole models.UserRole, memberID string) error {
	// Check if user is leaving (removing themselves)
	if requesterID == memberID {
		return s.repos.Member.Remove(ctx, conversationID, memberID)
	}

	// Check requester is owner/admin OR has global permission for removing others
	requester, err := s.repos.Member.GetByConversationAndUser(ctx, conversationID, requesterID)
	if err != nil {
		return errors.New("you are not a member of this conversation")
	}

	if !canModerateConversation(requester.MemberRole) {
		return errors.New("only owner, admin, or assistants can remove members")
	}

	// Perform removal
	if err := s.repos.Member.Remove(ctx, conversationID, memberID); err != nil {
		return err
	}

	// ---------------------------------------------------------
	// Send System Message (Async)
	// ---------------------------------------------------------
	go func() {
		// Fetch names
		ids := []string{memberID}
		if requesterID != memberID {
			ids = append(ids, requesterID)
		}

		users, err := s.authClient.GetBatchUsers(context.Background(), ids)
		if err != nil {
			fmt.Printf("[RemoveMember] Failed to fetch names: %v\n", err)
			return
		}

		memberName := "Unknown User"
		if u, ok := users[memberID]; ok {
			memberName = u.Name
		}

		var content string
		if requesterID == memberID {
			content = fmt.Sprintf("%s left the group", memberName)
		} else {
			requesterName := "Unknown User"
			if u, ok := users[requesterID]; ok {
				requesterName = u.Name
			}
			content = fmt.Sprintf("%s was removed by %s", memberName, requesterName)
		}

		// Create System Message
		sysMsg := &models.Message{
			ID:             uuid.New().String(),
			ConversationID: conversationID,
			SenderID:       "00000000-0000-0000-0000-000000000000", // Fix: Use valid UUID for system sender
			SenderRole:     "SYSTEM",
			Type:           models.MessageTypeSystem,
			Content:        content,
			MediaMetadata:  json.RawMessage("{}"),
		}

		if err := s.repos.Message.Create(context.Background(), sysMsg); err != nil {
			fmt.Printf("[RemoveMember] Failed to create system message: %v\n", err)
			return
		}

		// Notify members
		// TODO: This logic duplicates MessageService.notifyMembers.
		// Refactoring to a shared helper or using MessageService would be cleaner,
		// but simplified here to avoid circular dependency for now.
		conv, _ := s.repos.Conversation.GetByID(context.Background(), conversationID)
		if conv != nil {
			// Get remaining members
			memberIDs, _ := s.repos.Member.GetConversationMemberIDs(context.Background(), conversationID)
			if len(memberIDs) > 0 {
				s.notification.SendChatNotification(context.Background(), sysMsg, conv, memberIDs)
			}
		}
	}()

	return nil
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

// UpdateImage updates the conversation's profile image
func (s *ConversationService) UpdateImage(ctx context.Context, conversationID, userID string, userRole models.UserRole, imageURL string) error {
	// Check if user is a member and get their role
	member, err := s.repos.Member.GetByConversationAndUser(ctx, conversationID, userID)
	if err != nil {
		return errors.New("you are not a member of this conversation")
	}

	// Check permission (Owner, Admin, or high-level global role)
	if !canModerateConversation(member.MemberRole) {
		return errors.New("only owner, admin, or assistants can update group profile image")
	}

	return s.repos.Conversation.UpdateImage(ctx, conversationID, imageURL)
}

// IsMember checks if a user is a member of a conversation
func (s *ConversationService) IsMember(ctx context.Context, conversationID, userID string) (bool, error) {
	return s.repos.Member.IsMember(ctx, conversationID, userID)
}

func canModerateConversation(memberRole models.MemberRole) bool {
	// Moderation powers are strictly restricted to Group Owner and assigned Admins.
	return memberRole == models.MemberRoleOwner || memberRole == models.MemberRoleAdmin
}

// DeleteConversation deletes a conversation and all its associated data (messages, members, pins)
func (s *ConversationService) DeleteConversation(ctx context.Context, conversationID, userID string) error {
	// Check if user is a member and get their role
	member, err := s.repos.Member.GetByConversationAndUser(ctx, conversationID, userID)
	if err != nil {
		return errors.New("you are not a member of this conversation")
	}

	// Only Owner or Admin can delete the whole group
	if member.MemberRole != models.MemberRoleOwner && member.MemberRole != models.MemberRoleAdmin {
		return errors.New("only group owner or admin can delete the conversation")
	}

	// 0. Fetch all media URLs to delete from Cloudinary
	mediaURLs, err := s.repos.Message.GetAllMediaURLsInConversation(ctx, conversationID)
	if err == nil && len(mediaURLs) > 0 {
		// Delete media from Cloudinary (sequentially for safety, or we could go routine it)
		// Since the whole group is being deleted, we can do this in the background
		go func(urls []string) {
			for _, u := range urls {
				_ = s.media.DeleteMedia(context.Background(), u)
			}
		}(mediaURLs)
	}

	// 1. Delete all messages and pins
	if err := s.repos.Message.DeleteAllByConversation(ctx, conversationID); err != nil {
		return fmt.Errorf("failed to delete messages: %v", err)
	}

	// 2. Delete all members
	if err := s.repos.Member.DeleteAllByConversation(ctx, conversationID); err != nil {
		return fmt.Errorf("failed to delete members: %v", err)
	}

	// 3. Delete the conversation itself
	if err := s.repos.Conversation.Delete(ctx, conversationID); err != nil {
		return fmt.Errorf("failed to delete conversation: %v", err)
	}

	return nil
}

// GetMember retrieves a member from a conversation
func (s *ConversationService) GetMember(ctx context.Context, conversationID, userID string) (*models.ConversationMember, error) {
	return s.repos.Member.GetByConversationAndUser(ctx, conversationID, userID)
}

// canCreateGroup checks if a user role can create group chats
func canCreateGroup(role models.UserRole) bool {
	return role == models.UserRoleInstructor || role == models.UserRoleTeacher
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

// cacheConversationMembers stores member IDs in Redis for quick access
// This is used by the typing service to get recipients for typing events
func (s *ConversationService) cacheConversationMembers(ctx context.Context, conversationID string, memberIDs []string) {
	if s.redis == nil || len(memberIDs) == 0 {
		return
	}

	membersKey := fmt.Sprintf("conv:members:%s", conversationID)
	
	// Convert []string to []interface{} for Redis SAdd
	members := make([]interface{}, len(memberIDs))
	for i, id := range memberIDs {
		members[i] = id
	}

	// Add members to set
	if err := s.redis.SAdd(ctx, membersKey, members...); err != nil {
		fmt.Printf("[ConversationService] Failed to cache members: %v\n", err)
		return
	}

	// Set expiration (30 days)
	if err := s.redis.Expire(ctx, membersKey, 30*24*60*60); err != nil {
		fmt.Printf("[ConversationService] Failed to set expiration on members cache: %v\n", err)
	}
}
