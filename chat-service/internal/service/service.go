package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/graduation/chat-service/internal/events"
	"github.com/graduation/chat-service/internal/kafka"
	"github.com/graduation/chat-service/internal/models"
	"github.com/graduation/chat-service/internal/repository"
	"github.com/lib/pq"
)

type Service struct {
	repo        *repository.Repository
	producer    *kafka.Producer
	media       *MediaService
	userService *UserService
}

func NewService(repo *repository.Repository, producer *kafka.Producer, media *MediaService, userService *UserService) *Service {
	return &Service{
		repo:        repo,
		producer:    producer,
		media:       media,
		userService: userService,
	}
}

// --- Conversations ---

func (s *Service) CreateDirectConversation(initiatorID, peerID string) (*models.Conversation, error) {
	// TODO: Check if one already exists
	conv := &models.Conversation{
		Type:      models.Direct,
		CreatedBy: initiatorID,
		Members: []models.ConversationMember{
			{UserID: initiatorID, Role: models.RoleOwner, JoinedAt: time.Now()},
			{UserID: peerID, Role: models.RoleMember, JoinedAt: time.Now()},
		},
	}

	if err := s.repo.CreateConversation(conv); err != nil {
		return nil, err
	}
	return conv, nil
}

func (s *Service) CreateGroupConversation(creatorID, name string, memberIDs []string) (*models.Conversation, error) {
	members := make([]models.ConversationMember, len(memberIDs)+1)
	members[0] = models.ConversationMember{UserID: creatorID, Role: models.RoleOwner, JoinedAt: time.Now()}

	for i, uid := range memberIDs {
		members[i+1] = models.ConversationMember{UserID: uid, Role: models.RoleMember, JoinedAt: time.Now()}
	}

	conv := &models.Conversation{
		Type:      models.Group,
		Name:      name,
		CreatedBy: creatorID,
		Members:   members,
	}

	if err := s.repo.CreateConversation(conv); err != nil {
		return nil, err
	}
	return conv, nil
}

func (s *Service) GetUserConversations(userID string) ([]models.ConversationResponse, error) {
	convs, err := s.repo.GetUserConversations(userID)
	if err != nil {
		return nil, err
	}

	// Collect all user IDs to fetch profiles
	userIDsMap := make(map[string]bool)
	for _, conv := range convs {
		userIDsMap[conv.CreatedBy] = true
		// Get members for each conversation
		members, _ := s.repo.GetMembers(conv.ID)
		for _, m := range members {
			userIDsMap[m.UserID] = true
		}
	}

	// Convert map to slice
	userIDs := make([]string, 0, len(userIDsMap))
	for id := range userIDsMap {
		userIDs = append(userIDs, id)
	}

	// Fetch all user profiles
	profiles, err := s.userService.FetchUserProfiles(userIDs)
	if err != nil {
		fmt.Printf("Error fetching user profiles: %v\n", err)
		profiles = make(map[string]*UserProfile)
	}

	// Build enriched responses
	responses := make([]models.ConversationResponse, len(convs))
	for i, conv := range convs {
		responses[i] = models.ConversationResponse{
			ID:        conv.ID,
			Type:      conv.Type,
			Name:      conv.Name,
			ImageURL:  conv.ImageURL,
			CreatedBy: conv.CreatedBy,
			CreatedAt: conv.CreatedAt,
			UpdatedAt: conv.UpdatedAt,
		}

		// Get last message
		messages, _ := s.repo.GetMessages(conv.ID, 1, 0)
		if len(messages) > 0 {
			lastMsg := messages[0]
			senderProfile := profiles[lastMsg.SenderID]
			responses[i].LastMessage = &models.MessageResponse{
				ID:             lastMsg.ID,
				ConversationID: lastMsg.ConversationID,
				SenderID:       lastMsg.SenderID,
				Content:        lastMsg.Content,
				Type:           lastMsg.Type,
				MediaURLs:      lastMsg.MediaURLs,
				CreatedAt:      lastMsg.CreatedAt,
				Sender: &models.UserStub{
					ID:    lastMsg.SenderID,
					Name:  senderProfile.Name,
					Image: senderProfile.Image,
				},
			}
		}

		// For direct chats, show peer profile
		if conv.Type == models.Direct {
			members, _ := s.repo.GetMembers(conv.ID)
			for _, m := range members {
				if m.UserID != userID {
					peerProfile := profiles[m.UserID]
					responses[i].PeerProfile = &models.UserStub{
						ID:    m.UserID,
						Name:  peerProfile.Name,
						Image: peerProfile.Image,
					}
					// Use peer's name and avatar for direct chat display
					responses[i].Name = peerProfile.Name
					responses[i].ImageURL = peerProfile.Image
					break
				}
			}
		}
	}

	return responses, nil
}

func (s *Service) GetConversation(id, userID string) (*models.ConversationResponse, error) {
	isMember, err := s.repo.IsMember(id, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, errors.New("forbidden")
	}
	
	conv, err := s.repo.GetConversationByID(id)
	if err != nil {
		return nil, err
	}

	// Get members
	members, _ := s.repo.GetMembers(id)
	
	// Collect user IDs
	userIDs := []string{conv.CreatedBy}
	for _, m := range members {
		userIDs = append(userIDs, m.UserID)
	}

	// Fetch profiles
	profiles, _ := s.userService.FetchUserProfiles(userIDs)

	// Build enriched response
	response := &models.ConversationResponse{
		ID:        conv.ID,
		Type:      conv.Type,
		Name:      conv.Name,
		ImageURL:  conv.ImageURL,
		CreatedBy: conv.CreatedBy,
		CreatedAt: conv.CreatedAt,
		UpdatedAt: conv.UpdatedAt,
	}

	// Enrich members
	enrichedMembers := make([]models.ConversationMemberResponse, len(members))
	for i, m := range members {
		profile := profiles[m.UserID]
		enrichedMembers[i] = models.ConversationMemberResponse{
			ConversationID: m.ConversationID,
			UserID:         m.UserID,
			Role:           m.Role,
			JoinedAt:       m.JoinedAt,
			LastReadAt:     m.LastReadAt,
			Profile: &models.UserStub{
				ID:    m.UserID,
				Name:  profile.Name,
				Image: profile.Image,
			},
		}
	}
	response.Members = enrichedMembers

	// For direct chats, show peer profile
	if conv.Type == models.Direct {
		for _, m := range members {
			if m.UserID != userID {
				peerProfile := profiles[m.UserID]
				response.PeerProfile = &models.UserStub{
					ID:    m.UserID,
					Name:  peerProfile.Name,
					Image: peerProfile.Image,
				}
				response.Name = peerProfile.Name
				response.ImageURL = peerProfile.Image
				break
			}
		}
	}

	return response, nil
}

// --- Messages ---

func (s *Service) SendMessage(conversationID, senderID string, content string, msgType models.MessageType, media pq.StringArray) (*models.Message, error) {
	// 1. Verify Membership
	isMember, _ := s.repo.IsMember(conversationID, senderID)
	if !isMember {
		return nil, errors.New("forbidden")
	}

	// 2. Create Message
	msg := &models.Message{
		ConversationID: conversationID,
		SenderID:       senderID,
		Content:        content,
		Type:           msgType,
		MediaURLs:      media,
	}

	if err := s.repo.CreateMessage(msg); err != nil {
		return nil, err
	}

	// 3. Fetch Recipients (CRITICAL)
	members, err := s.repo.GetMembers(conversationID)
	if err != nil {
		// Log error but assume persistence worked? No, we need members for notifications.
		return msg, nil
	}

	recipientIDs := make([]string, 0, len(members))
	for _, m := range members {
		recipientIDs = append(recipientIDs, m.UserID)
	}

	// 4. Produce Event
	event := events.MessageCreatedEvent{
		ID:             msg.ID,
		ConversationID: msg.ConversationID,
		SenderID:       msg.SenderID,
		Content:        msg.Content,
		Type:           string(msg.Type),
		MediaURLs:      []string(msg.MediaURLs), // Convert pq.StringArray to []string
		CreatedAt:      msg.CreatedAt,
		RecipientIDs:   recipientIDs, // <--- Correctly populated
	}

	if err := s.producer.PublishMessageCreated(event); err != nil {
		fmt.Printf("Kafka Publish Error: %v\n", err)
	} else {
		fmt.Printf("Kafka Published! %s -> %s\n", msg.ID, event.RecipientIDs)
	}

	return msg, nil
}

func (s *Service) GetMessages(conversationID, userID string, limit, offset int) ([]models.MessageResponse, error) {
	isMember, _ := s.repo.IsMember(conversationID, userID)
	if !isMember {
		return nil, errors.New("forbidden")
	}
	
	messages, err := s.repo.GetMessages(conversationID, limit, offset)
	if err != nil {
		return nil, err
	}

	// Collect sender IDs
	senderIDs := make([]string, 0, len(messages))
	senderIDsMap := make(map[string]bool)
	for _, msg := range messages {
		if !senderIDsMap[msg.SenderID] {
			senderIDs = append(senderIDs, msg.SenderID)
			senderIDsMap[msg.SenderID] = true
		}
	}

	// Fetch sender profiles
	profiles, _ := s.userService.FetchUserProfiles(senderIDs)

	// Build enriched responses
	responses := make([]models.MessageResponse, len(messages))
	for i, msg := range messages {
		senderProfile := profiles[msg.SenderID]
		responses[i] = models.MessageResponse{
			ID:             msg.ID,
			ConversationID: msg.ConversationID,
			SenderID:       msg.SenderID,
			Content:        msg.Content,
			Type:           msg.Type,
			MediaURLs:      msg.MediaURLs,
			CreatedAt:      msg.CreatedAt,
			Sender: &models.UserStub{
				ID:    msg.SenderID,
				Name:  senderProfile.Name,
				Image: senderProfile.Image,
			},
		}
	}

	return responses, nil
}

func (s *Service) SetTyping(conversationID, userID string, isTyping bool) error {
	// 1. Verify Member
	isMember, _ := s.repo.IsMember(conversationID, userID)
	if !isMember {
		return errors.New("forbidden")
	}

	// 2. Fetch Recipients
	members, err := s.repo.GetMembers(conversationID)
	if err != nil {
		return err
	}

	recipientIDs := make([]string, 0, len(members))
	for _, m := range members {
		// Don't send typing event to self? Debatable, but usually fine to filter or include.
		// Let's include all, Gateway filters sender usually.
		recipientIDs = append(recipientIDs, m.UserID)
	}

	// 3. Produce Event
	event := events.TypingEvent{
		ConversationID: conversationID,
		UserID:         userID,
		IsTyping:       isTyping,
		RecipientIDs:   recipientIDs,
	}

	return s.producer.PublishTyping(event)
}

// --- Pinning ---

func (s *Service) PinMessage(conversationID, messageID, userID string) error {
	// Verify Member & Permission (Owner/Admin)
	// Simplified: Check if member first
	isMember, _ := s.repo.IsMember(conversationID, userID)
	if !isMember {
		return errors.New("forbidden")
	}

	pm := &models.PinnedMessage{
		ConversationID: conversationID,
		MessageID:      messageID,
		PinnedBy:       userID,
	}
	return s.repo.PinMessage(pm)
}

func (s *Service) UnpinMessage(conversationID, messageID, userID string) error {
	isMember, _ := s.repo.IsMember(conversationID, userID)
	if !isMember {
		return errors.New("forbidden")
	}
	return s.repo.UnpinMessage(conversationID, messageID)
}

func (s *Service) GetPinnedMessages(conversationID, userID string) ([]models.PinnedMessage, error) {
	isMember, _ := s.repo.IsMember(conversationID, userID)
	if !isMember {
		return nil, errors.New("forbidden")
	}
	return s.repo.GetPinnedMessages(conversationID)
}

// --- Media ---

func (s *Service) PresignMedia(folder string) map[string]string {
	return s.media.GeneratePresignedURL(folder)
}

// --- Member Management ---

func (s *Service) AddMember(conversationID, requesterID, newMemberID string) error {
	// 1. Verify requester is owner/admin
	member, err := s.repo.GetMember(conversationID, requesterID)
	if err != nil {
		return errors.New("forbidden")
	}
	if member.Role != models.RoleOwner && member.Role != models.RoleAdmin {
		return errors.New("only owners and admins can add members")
	}

	// 2. Check if already a member
	isMember, _ := s.repo.IsMember(conversationID, newMemberID)
	if isMember {
		return errors.New("user is already a member")
	}

	// 3. Add member
	newMember := &models.ConversationMember{
		ConversationID: conversationID,
		UserID:         newMemberID,
		Role:           models.RoleMember,
		JoinedAt:       time.Now(),
	}
	return s.repo.AddMember(newMember)
}

func (s *Service) RemoveMember(conversationID, requesterID, targetUserID string) error {
	// 1. Verify requester is owner/admin
	member, err := s.repo.GetMember(conversationID, requesterID)
	if err != nil {
		return errors.New("forbidden")
	}
	if member.Role != models.RoleOwner && member.Role != models.RoleAdmin {
		return errors.New("only owners and admins can remove members")
	}

	// 2. Cannot remove owner
	targetMember, err := s.repo.GetMember(conversationID, targetUserID)
	if err != nil {
		return errors.New("member not found")
	}
	if targetMember.Role == models.RoleOwner {
		return errors.New("cannot remove owner")
	}

	// 3. Remove member
	return s.repo.RemoveMember(conversationID, targetUserID)
}

func (s *Service) UpdateMemberRole(conversationID, requesterID, targetUserID string, newRole models.MemberRole) error {
	// 1. Verify requester is owner
	member, err := s.repo.GetMember(conversationID, requesterID)
	if err != nil {
		return errors.New("forbidden")
	}
	if member.Role != models.RoleOwner {
		return errors.New("only owner can update roles")
	}

	// 2. Cannot change owner role
	targetMember, err := s.repo.GetMember(conversationID, targetUserID)
	if err != nil {
		return errors.New("member not found")
	}
	if targetMember.Role == models.RoleOwner {
		return errors.New("cannot change owner role")
	}

	// 3. Update role
	return s.repo.UpdateMemberRole(conversationID, targetUserID, newRole)
}

func (s *Service) GetMembers(conversationID, userID string) ([]models.ConversationMemberResponse, error) {
	// Verify membership
	isMember, _ := s.repo.IsMember(conversationID, userID)
	if !isMember {
		return nil, errors.New("forbidden")
	}
	
	members, err := s.repo.GetMembers(conversationID)
	if err != nil {
		return nil, err
	}

	// Collect user IDs
	userIDs := make([]string, len(members))
	for i, m := range members {
		userIDs[i] = m.UserID
	}

	// Fetch profiles
	profiles, _ := s.userService.FetchUserProfiles(userIDs)

	// Build enriched responses
	responses := make([]models.ConversationMemberResponse, len(members))
	for i, m := range members {
		profile := profiles[m.UserID]
		responses[i] = models.ConversationMemberResponse{
			ConversationID: m.ConversationID,
			UserID:         m.UserID,
			Role:           m.Role,
			JoinedAt:       m.JoinedAt,
			LastReadAt:     m.LastReadAt,
			Profile: &models.UserStub{
				ID:    m.UserID,
				Name:  profile.Name,
				Image: profile.Image,
			},
		}
	}

	return responses, nil
}

func (s *Service) LeaveConversation(conversationID, userID string) error {
	// 1. Verify membership
	member, err := s.repo.GetMember(conversationID, userID)
	if err != nil {
		return errors.New("not a member")
	}

	// 2. Owner cannot leave (must delete or transfer ownership)
	if member.Role == models.RoleOwner {
		return errors.New("owner cannot leave, must delete conversation or transfer ownership")
	}

	// 3. Remove member
	return s.repo.RemoveMember(conversationID, userID)
}

func (s *Service) DeleteConversation(conversationID, userID string) error {
	// Get conversation type
	conv, err := s.repo.GetConversationByID(conversationID)
	if err != nil {
		return errors.New("conversation not found")
	}

	// Check permissions based on conversation type
	if conv.Type == models.Direct {
		// For direct chats, any member can delete
		isMember, _ := s.repo.IsMember(conversationID, userID)
		if !isMember {
			return errors.New("forbidden")
		}
	} else {
		// For groups, only owner/admin can delete
		member, err := s.repo.GetMember(conversationID, userID)
		if err != nil {
			return errors.New("forbidden")
		}
		if member.Role != models.RoleOwner && member.Role != models.RoleAdmin {
			return errors.New("only owner or admin can delete group conversation")
		}
	}

	// Delete conversation
	return s.repo.DeleteConversation(conversationID)
}
