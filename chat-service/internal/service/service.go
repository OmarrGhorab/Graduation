package service

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/graduation/chat-service/internal/events"
	"github.com/graduation/chat-service/internal/kafka"
	"github.com/graduation/chat-service/internal/models"
	"github.com/graduation/chat-service/internal/repository"
	"github.com/lib/pq"
)

type Service struct {
	repo                *repository.Repository
	producer            *kafka.Producer
	media               *MediaService
	userService         *UserService
	presenceService     *PresenceService
	notificationService *NotificationService
}

func NewService(repo *repository.Repository, producer *kafka.Producer, media *MediaService, userService *UserService, presenceService *PresenceService, notificationService *NotificationService) *Service {
	return &Service{
		repo:                repo,
		producer:            producer,
		media:               media,
		userService:         userService,
		presenceService:     presenceService,
		notificationService: notificationService,
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

func (s *Service) CreateGroupConversation(creatorID, name, description string, memberIDs []string) (*models.Conversation, error) {
	members := make([]models.ConversationMember, len(memberIDs)+1)
	members[0] = models.ConversationMember{UserID: creatorID, Role: models.RoleOwner, JoinedAt: time.Now()}

	for i, uid := range memberIDs {
		members[i+1] = models.ConversationMember{UserID: uid, Role: models.RoleMember, JoinedAt: time.Now()}
	}

	conv := &models.Conversation{
		Type:        models.Group,
		Name:        name,
		Description: description,
		CreatedBy:   creatorID,
		Members:     members,
	}

	if err := s.repo.CreateConversation(conv); err != nil {
		return nil, err
	}
	return conv, nil
}

func (s *Service) GetUserConversations(userID, convType, search string, limit, offset int) ([]models.ConversationResponse, error) {
	convs, err := s.repo.GetUserConversations(userID)
	if err != nil {
		return nil, err
	}

	// Filter by type if specified
	if convType != "" {
		filtered := []models.Conversation{}
		for _, conv := range convs {
			if string(conv.Type) == convType {
				filtered = append(filtered, conv)
			}
		}
		convs = filtered
	}

	// Filter by search if specified
	if search != "" {
		filtered := []models.Conversation{}
		searchLower := strings.ToLower(search)
		for _, conv := range convs {
			if strings.Contains(strings.ToLower(conv.Name), searchLower) ||
				strings.Contains(strings.ToLower(conv.Description), searchLower) {
				filtered = append(filtered, conv)
			}
		}
		convs = filtered
	}

	// Sort by updated_at descending (most recent first) after filtering
	// This ensures conversations with latest messages appear first
	sort.Slice(convs, func(i, j int) bool {
		return convs[i].UpdatedAt.After(convs[j].UpdatedAt)
	})

	// Apply pagination
	total := len(convs)
	if offset >= total {
		return []models.ConversationResponse{}, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	convs = convs[offset:end]

	// Collect all user IDs to fetch profiles and presence
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

	// Fetch presence information
	presenceMap, err := s.presenceService.CheckPresence(userIDs)
	if err != nil {
		fmt.Printf("Error fetching presence: %v\n", err)
		presenceMap = make(map[string]bool)
	}

	// Build enriched responses
	responses := make([]models.ConversationResponse, len(convs))
	for i, conv := range convs {
		responses[i] = models.ConversationResponse{
			ID:          conv.ID,
			Type:        conv.Type,
			Name:        conv.Name,
			Description: conv.Description,
			ImageURL:    conv.ImageURL,
			CreatedBy:   conv.CreatedBy,
			CreatedAt:   conv.CreatedAt,
			UpdatedAt:   conv.UpdatedAt,
		}

		// Get unread count for this conversation
		unreadCount, _ := s.repo.GetUnreadCount(conv.ID, userID)
		responses[i].UnreadCount = int(unreadCount)

		// Get last message
		messages, _ := s.repo.GetMessages(conv.ID, "", 1, 0)
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
				ReplyToID:      lastMsg.ReplyToID,
				IsDeleted:      lastMsg.IsDeleted,
				CreatedAt:      lastMsg.CreatedAt,
				Sender: &models.UserStub{
					ID:    lastMsg.SenderID,
					Name:  senderProfile.Name,
					Image: senderProfile.Image,
				},
			}
		}

		// For direct chats, show peer profile and online status
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
					responses[i].PeerOnline = presenceMap[m.UserID]
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

	// Fetch presence
	presenceMap, _ := s.presenceService.CheckPresence(userIDs)

	// Build enriched response
	response := &models.ConversationResponse{
		ID:          conv.ID,
		Type:        conv.Type,
		Name:        conv.Name,
		Description: conv.Description,
		ImageURL:    conv.ImageURL,
		CreatedBy:   conv.CreatedBy,
		CreatedAt:   conv.CreatedAt,
		UpdatedAt:   conv.UpdatedAt,
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
			IsOnline: presenceMap[m.UserID],
		}
	}
	response.Members = enrichedMembers

	// For direct chats, show peer profile and online status
	if conv.Type == models.Direct {
		for _, m := range members {
			if m.UserID != userID {
				peerProfile := profiles[m.UserID]
				response.PeerProfile = &models.UserStub{
					ID:    m.UserID,
					Name:  peerProfile.Name,
					Image: peerProfile.Image,
				}
				response.PeerOnline = presenceMap[m.UserID]
				response.Name = peerProfile.Name
				response.ImageURL = peerProfile.Image
				break
			}
		}
	}

	return response, nil
}

// --- Messages ---

func (s *Service) SendMessage(conversationID, senderID string, content string, msgType models.MessageType, media pq.StringArray, replyToID *string) (*models.Message, error) {
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
		ReplyToID:      replyToID,
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
	unreadCounts := make(map[string]int)
	for _, m := range members {
		recipientIDs = append(recipientIDs, m.UserID)
		// Fetch unread count for this user (for real-time WS update)
		count, _ := s.repo.GetUnreadCount(conversationID, m.UserID)
		unreadCounts[m.UserID] = int(count)
	}

	// 4. Produce Event
	event := events.MessageCreatedEvent{
		ID:                    msg.ID,
		ConversationID:        msg.ConversationID,
		SenderID:              msg.SenderID,
		Content:               msg.Content,
		Type:                  string(msg.Type),
		MediaURLs:             []string(msg.MediaURLs),
		CreatedAt:             msg.CreatedAt,
		RecipientIDs:          recipientIDs,
		RecipientUnreadCounts: unreadCounts,
	}

	if err := s.producer.PublishMessageCreated(event); err != nil {
		fmt.Printf("Kafka Publish Error: %v\n", err)
	} else {
		fmt.Printf("Kafka Published! %s -> %s\n", msg.ID, event.RecipientIDs)
	}

	// 5. Send notifications to offline users
	fmt.Printf("[DEBUG] Starting offline notification check for message %s\n", msg.ID)
	fmt.Printf("[DEBUG] Total members in conversation: %d\n", len(members))
	go s.sendOfflineNotifications(msg, members)

	return msg, nil
}

// sendOfflineNotifications sends push notifications to offline users
func (s *Service) sendOfflineNotifications(msg *models.Message, members []models.ConversationMember) {
	fmt.Printf("[DEBUG] sendOfflineNotifications called for message %s\n", msg.ID)

	// Get online users from presence service
	userIDs := make([]string, len(members))
	for i, m := range members {
		userIDs[i] = m.UserID
	}

	fmt.Printf("[DEBUG] Checking presence for %d users: %v\n", len(userIDs), userIDs)
	onlineUsers, err := s.presenceService.CheckPresence(userIDs)
	if err != nil {
		fmt.Printf("[ERROR] Failed to check presence: %v\n", err)
		// Continue anyway - better to send notifications than fail silently
		onlineUsers = make(map[string]bool)
	}

	fmt.Printf("[DEBUG] Presence check result: %v\n", onlineUsers)

	// Get sender profile
	profiles, _ := s.userService.FetchUserProfiles([]string{msg.SenderID})
	senderProfile := profiles[msg.SenderID]

	// Handle nil profile gracefully
	senderName := "Unknown User"
	senderImage := ""
	if senderProfile != nil {
		senderName = senderProfile.Name
		senderImage = senderProfile.Image
	}

	// Get conversation details
	conv, _ := s.repo.GetConversationByID(msg.ConversationID)
	conversationName := ""
	if conv != nil {
		if conv.Type == models.Group {
			conversationName = conv.Name
		} else {
			conversationName = senderName
		}
	}

	// Send notification to all users (except sender)
	// FCM is sent even to online users because:
	// 1. App might be in background (user won't see WebSocket message)
	// 2. WebSocket might be unstable
	// 3. Better to have redundancy than miss notifications
	notificationCount := 0
	for _, member := range members {
		// Skip sender
		if member.UserID == msg.SenderID {
			fmt.Printf("[DEBUG] Skipping sender %s\n", member.UserID)
			continue
		}

		// Check online status for logging purposes only
		isOnline := onlineUsers[member.UserID]
		if isOnline {
			fmt.Printf("[DEBUG] User %s is ONLINE but sending FCM anyway (app might be in background)\n", member.UserID)
		} else {
			fmt.Printf("[DEBUG] User %s is OFFLINE - sending FCM notification\n", member.UserID)
		}
		notificationCount++

		// Prepare notification content
		notificationBody := msg.Content
		if msg.Type == models.Image {
			notificationBody = "📷 Photo"
		} else if msg.Type == models.Voice {
			notificationBody = "🎤 Voice message"
		}

		// Get updated unread count for this specific user
		unreadCount, _ := s.repo.GetUnreadCount(msg.ConversationID, member.UserID)

		// Send push notification
		payload := NotificationPayload{
			UserID: member.UserID,
			Type:   "chat.message",
			Data: map[string]interface{}{
				"message_id":        msg.ID,
				"conversation_id":   msg.ConversationID,
				"conversation_name": conversationName,
				"sender_id":         msg.SenderID,
				"sender_name":       senderName,
				"sender_image":      senderImage,
				"content":           msg.Content,
				"body":              notificationBody,
				"type":              string(msg.Type),
				"unread_count":      fmt.Sprintf("%d", unreadCount),
				"created_at":        msg.CreatedAt.Format(time.RFC3339),
			},
		}

		fmt.Printf("[DEBUG] Sending notification to user %s via notification service\n", member.UserID)
		if err := s.notificationService.SendNotification(payload); err != nil {
			fmt.Printf("[ERROR] Failed to send notification to user %s: %v\n", member.UserID, err)
		} else {
			fmt.Printf("[SUCCESS] FCM notification sent to user %s\n", member.UserID)
		}
	}

	fmt.Printf("[SUMMARY] FCM notifications sent: %d, total members: %d\n", notificationCount, len(members))
}

func (s *Service) GetMessages(conversationID, userID, search string, limit, offset int) ([]models.MessageResponse, error) {
	isMember, _ := s.repo.IsMember(conversationID, userID)
	if !isMember {
		return nil, errors.New("forbidden")
	}

	messages, err := s.repo.GetMessages(conversationID, search, limit, offset)
	if err != nil {
		return nil, err
	}

	return s.enrichMessages(messages)
}

// Helper method to enrich a single message
func (s *Service) EnrichMessageResponse(msg *models.Message) (*models.MessageResponse, error) {
	messages := []models.Message{*msg}
	enriched, err := s.enrichMessages(messages)
	if err != nil {
		return nil, err
	}
	if len(enriched) == 0 {
		return nil, errors.New("failed to enrich message")
	}
	return &enriched[0], nil
}

// Helper method to enrich multiple messages with sender and reply_to details
func (s *Service) enrichMessages(messages []models.Message) ([]models.MessageResponse, error) {
	if len(messages) == 0 {
		return []models.MessageResponse{}, nil
	}

	// Collect sender IDs and reply_to message IDs
	senderIDs := make([]string, 0, len(messages))
	senderIDsMap := make(map[string]bool)
	replyToIDs := make([]string, 0)
	replyToIDsMap := make(map[string]bool)

	for _, msg := range messages {
		if !senderIDsMap[msg.SenderID] {
			senderIDs = append(senderIDs, msg.SenderID)
			senderIDsMap[msg.SenderID] = true
		}
		if msg.ReplyToID != nil && *msg.ReplyToID != "" && !replyToIDsMap[*msg.ReplyToID] {
			replyToIDs = append(replyToIDs, *msg.ReplyToID)
			replyToIDsMap[*msg.ReplyToID] = true
		}
	}

	// Fetch sender profiles
	profiles, _ := s.userService.FetchUserProfiles(senderIDs)

	// Fetch reply_to messages if any
	replyToMessages := make(map[string]*models.Message)
	if len(replyToIDs) > 0 {
		for _, replyID := range replyToIDs {
			replyMsg, err := s.repo.GetMessageByID(replyID)
			if err == nil {
				replyToMessages[replyID] = replyMsg
				// Also fetch the sender profile for the reply message
				if !senderIDsMap[replyMsg.SenderID] {
					senderIDs = append(senderIDs, replyMsg.SenderID)
					senderIDsMap[replyMsg.SenderID] = true
				}
			}
		}
		// Fetch profiles for reply message senders
		if len(senderIDs) > len(profiles) {
			profiles, _ = s.userService.FetchUserProfiles(senderIDs)
		}
	}

	// Build enriched responses
	responses := make([]models.MessageResponse, len(messages))
	for i, msg := range messages {
		senderProfile := profiles[msg.SenderID]

		// Handle nil profile gracefully
		senderName := "Unknown User"
		senderImage := ""
		if senderProfile != nil {
			senderName = senderProfile.Name
			senderImage = senderProfile.Image
		}

		responses[i] = models.MessageResponse{
			ID:             msg.ID,
			ConversationID: msg.ConversationID,
			SenderID:       msg.SenderID,
			Content:        msg.Content,
			Type:           msg.Type,
			MediaURLs:      msg.MediaURLs,
			ReplyToID:      msg.ReplyToID,
			IsDeleted:      msg.IsDeleted,
			CreatedAt:      msg.CreatedAt,
			Sender: &models.UserStub{
				ID:    msg.SenderID,
				Name:  senderName,
				Image: senderImage,
			},
		}

		// Add reply_to details if exists
		if msg.ReplyToID != nil && *msg.ReplyToID != "" {
			if replyMsg, ok := replyToMessages[*msg.ReplyToID]; ok {
				replyProfile := profiles[replyMsg.SenderID]

				// Handle nil profile gracefully
				replyName := "Unknown User"
				replyImage := ""
				if replyProfile != nil {
					replyName = replyProfile.Name
					replyImage = replyProfile.Image
				}

				responses[i].ReplyTo = &models.MessageReplyToStub{
					ID:      replyMsg.ID,
					Content: replyMsg.Content,
					Sender: &models.UserStub{
						ID:    replyMsg.SenderID,
						Name:  replyName,
						Image: replyImage,
					},
				}
			}
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
		recipientIDs = append(recipientIDs, m.UserID)
	}

	// 3. Fetch User Profile for name
	userName := "Someone"
	profiles, err := s.userService.FetchUserProfiles([]string{userID})
	if err == nil && profiles[userID] != nil {
		userName = profiles[userID].Name
	}

	// 4. Produce Event
	event := events.TypingEvent{
		ConversationID: conversationID,
		UserID:         userID,
		UserName:       userName,
		IsTyping:       isTyping,
		RecipientIDs:   recipientIDs,
	}

	return s.producer.PublishTyping(event)
}

func (s *Service) DeleteMessage(conversationID, messageID, userID string) error {
	// 1. Verify membership
	isMember, _ := s.repo.IsMember(conversationID, userID)
	if !isMember {
		return errors.New("forbidden")
	}

	// 2. Verify message exists and belongs to conversation
	message, err := s.repo.GetMessageByID(messageID)
	if err != nil {
		return errors.New("message not found")
	}
	if message.ConversationID != conversationID {
		return errors.New("message does not belong to this conversation")
	}

	// 3. Only sender can delete their own message
	if message.SenderID != userID {
		return errors.New("only sender can delete their message")
	}

	// 4. Soft delete the message
	return s.repo.DeleteMessage(messageID)
}

// --- Pinning ---

func (s *Service) PinMessage(conversationID, messageID, userID string) (*models.PinnedMessageResponse, error) {
	// Verify Member & Permission (Owner/Admin)
	// Simplified: Check if member first
	isMember, _ := s.repo.IsMember(conversationID, userID)
	if !isMember {
		return nil, errors.New("forbidden")
	}

	pm := &models.PinnedMessage{
		ConversationID: conversationID,
		MessageID:      messageID,
		PinnedBy:       userID,
	}

	if err := s.repo.PinMessage(pm); err != nil {
		return nil, err
	}

	// Fetch the message
	message, err := s.repo.GetMessageByID(messageID)
	if err != nil {
		return nil, err
	}

	// Collect user IDs
	userIDs := []string{userID, message.SenderID}

	// Check for reply_to sender
	if message.ReplyToID != nil && *message.ReplyToID != "" {
		replyMsg, err := s.repo.GetMessageByID(*message.ReplyToID)
		if err == nil {
			userIDs = append(userIDs, replyMsg.SenderID)
		}
	}

	// Fetch profiles
	profiles, _ := s.userService.FetchUserProfiles(userIDs)

	// Build enriched response
	pinnerProfile := profiles[userID]
	senderProfile := profiles[message.SenderID]

	response := &models.PinnedMessageResponse{
		ID:             pm.ID,
		MessageID:      pm.MessageID,
		ConversationID: pm.ConversationID,
		PinnedBy:       pm.PinnedBy,
		PinnedAt:       pm.PinnedAt,
		Pinner: &models.UserStub{
			ID:    userID,
			Name:  pinnerProfile.Name,
			Image: pinnerProfile.Image,
		},
		Message: &models.MessageResponse{
			ID:             message.ID,
			ConversationID: message.ConversationID,
			SenderID:       message.SenderID,
			Content:        message.Content,
			Type:           message.Type,
			MediaURLs:      message.MediaURLs,
			ReplyToID:      message.ReplyToID,
			IsDeleted:      message.IsDeleted,
			CreatedAt:      message.CreatedAt,
			Sender: &models.UserStub{
				ID:    message.SenderID,
				Name:  senderProfile.Name,
				Image: senderProfile.Image,
			},
		},
	}

	// Add reply_to details if exists
	if message.ReplyToID != nil && *message.ReplyToID != "" {
		replyMsg, err := s.repo.GetMessageByID(*message.ReplyToID)
		if err == nil {
			replyProfile := profiles[replyMsg.SenderID]
			response.Message.ReplyTo = &models.MessageReplyToStub{
				ID:      replyMsg.ID,
				Content: replyMsg.Content,
				Sender: &models.UserStub{
					ID:    replyMsg.SenderID,
					Name:  replyProfile.Name,
					Image: replyProfile.Image,
				},
			}
		}
	}

	return response, nil
}

func (s *Service) UnpinMessage(conversationID, messageID, userID string) error {
	isMember, _ := s.repo.IsMember(conversationID, userID)
	if !isMember {
		return errors.New("forbidden")
	}
	return s.repo.UnpinMessage(conversationID, messageID)
}

func (s *Service) GetPinnedMessages(conversationID, userID string) ([]models.PinnedMessageResponse, error) {
	isMember, _ := s.repo.IsMember(conversationID, userID)
	if !isMember {
		return nil, errors.New("forbidden")
	}

	pins, err := s.repo.GetPinnedMessages(conversationID)
	if err != nil {
		return nil, err
	}

	if len(pins) == 0 {
		return []models.PinnedMessageResponse{}, nil
	}

	// Collect user IDs (senders and pinners)
	userIDsMap := make(map[string]bool)
	for _, pin := range pins {
		userIDsMap[pin.PinnedBy] = true
		if pin.Message != nil {
			userIDsMap[pin.Message.SenderID] = true
			// Also collect reply_to sender if exists
			if pin.Message.ReplyToID != nil && *pin.Message.ReplyToID != "" {
				replyMsg, err := s.repo.GetMessageByID(*pin.Message.ReplyToID)
				if err == nil {
					userIDsMap[replyMsg.SenderID] = true
				}
			}
		}
	}

	// Convert map to slice
	userIDs := make([]string, 0, len(userIDsMap))
	for id := range userIDsMap {
		userIDs = append(userIDs, id)
	}

	// Fetch profiles
	profiles, _ := s.userService.FetchUserProfiles(userIDs)

	// Build enriched responses
	responses := make([]models.PinnedMessageResponse, len(pins))
	for i, pin := range pins {
		pinnerProfile := profiles[pin.PinnedBy]

		responses[i] = models.PinnedMessageResponse{
			ID:             pin.ID,
			MessageID:      pin.MessageID,
			ConversationID: pin.ConversationID,
			PinnedBy:       pin.PinnedBy,
			PinnedAt:       pin.PinnedAt,
			Pinner: &models.UserStub{
				ID:    pin.PinnedBy,
				Name:  pinnerProfile.Name,
				Image: pinnerProfile.Image,
			},
		}

		// Enrich message if exists
		if pin.Message != nil {
			senderProfile := profiles[pin.Message.SenderID]
			enrichedMsg := &models.MessageResponse{
				ID:             pin.Message.ID,
				ConversationID: pin.Message.ConversationID,
				SenderID:       pin.Message.SenderID,
				Content:        pin.Message.Content,
				Type:           pin.Message.Type,
				MediaURLs:      pin.Message.MediaURLs,
				ReplyToID:      pin.Message.ReplyToID,
				IsDeleted:      pin.Message.IsDeleted,
				CreatedAt:      pin.Message.CreatedAt,
				Sender: &models.UserStub{
					ID:    pin.Message.SenderID,
					Name:  senderProfile.Name,
					Image: senderProfile.Image,
				},
			}

			// Add reply_to details if exists
			if pin.Message.ReplyToID != nil && *pin.Message.ReplyToID != "" {
				replyMsg, err := s.repo.GetMessageByID(*pin.Message.ReplyToID)
				if err == nil {
					replyProfile := profiles[replyMsg.SenderID]
					enrichedMsg.ReplyTo = &models.MessageReplyToStub{
						ID:      replyMsg.ID,
						Content: replyMsg.Content,
						Sender: &models.UserStub{
							ID:    replyMsg.SenderID,
							Name:  replyProfile.Name,
							Image: replyProfile.Image,
						},
					}
				}
			}

			responses[i].Message = enrichedMsg
		}
	}

	return responses, nil
}

// GetMediaCollection returns all media (photos, voice, links) from a conversation
func (s *Service) GetMediaCollection(conversationID, userID string) (*models.MediaCollectionResponse, error) {
	// Verify membership
	isMember, _ := s.repo.IsMember(conversationID, userID)
	if !isMember {
		return nil, errors.New("forbidden")
	}

	// Fetch photos
	photoMessages, _ := s.repo.GetMessagesByType(conversationID, models.Image)

	// Fetch voice messages
	voiceMessages, _ := s.repo.GetMessagesByType(conversationID, models.Voice)

	// Fetch messages with links
	linkMessages, _ := s.repo.GetMessagesWithLinks(conversationID)

	// Collect all sender IDs
	userIDsMap := make(map[string]bool)
	for _, msg := range photoMessages {
		userIDsMap[msg.SenderID] = true
	}
	for _, msg := range voiceMessages {
		userIDsMap[msg.SenderID] = true
	}
	for _, msg := range linkMessages {
		userIDsMap[msg.SenderID] = true
	}

	// Fetch profiles
	userIDs := make([]string, 0, len(userIDsMap))
	for id := range userIDsMap {
		userIDs = append(userIDs, id)
	}
	profiles, _ := s.userService.FetchUserProfiles(userIDs)

	// Build response
	response := &models.MediaCollectionResponse{
		Photos: make([]models.MediaItem, 0),
		Voice:  make([]models.MediaItem, 0),
		Links:  make([]models.MediaItem, 0),
	}

	// Process photos
	for _, msg := range photoMessages {
		senderProfile := profiles[msg.SenderID]
		for _, url := range msg.MediaURLs {
			response.Photos = append(response.Photos, models.MediaItem{
				MessageID: msg.ID,
				URL:       url,
				Type:      models.Image,
				SenderID:  msg.SenderID,
				Sender: &models.UserStub{
					ID:    msg.SenderID,
					Name:  senderProfile.Name,
					Image: senderProfile.Image,
				},
				CreatedAt: msg.CreatedAt,
			})
		}
	}

	// Process voice messages
	for _, msg := range voiceMessages {
		senderProfile := profiles[msg.SenderID]
		for _, url := range msg.MediaURLs {
			response.Voice = append(response.Voice, models.MediaItem{
				MessageID: msg.ID,
				URL:       url,
				Type:      models.Voice,
				SenderID:  msg.SenderID,
				Sender: &models.UserStub{
					ID:    msg.SenderID,
					Name:  senderProfile.Name,
					Image: senderProfile.Image,
				},
				CreatedAt: msg.CreatedAt,
			})
		}
	}

	// Process links (extract URLs from content)
	for _, msg := range linkMessages {
		senderProfile := profiles[msg.SenderID]
		// Simple URL extraction - in production, use a proper URL parser
		response.Links = append(response.Links, models.MediaItem{
			MessageID: msg.ID,
			URL:       msg.Content, // Full content with link
			Type:      models.Text,
			SenderID:  msg.SenderID,
			Sender: &models.UserStub{
				ID:    msg.SenderID,
				Name:  senderProfile.Name,
				Image: senderProfile.Image,
			},
			CreatedAt: msg.CreatedAt,
		})
	}

	return response, nil
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

	// Fetch presence
	presenceMap, _ := s.presenceService.CheckPresence(userIDs)

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
			IsOnline: presenceMap[m.UserID],
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

// --- Read Receipts ---

// MarkAsRead marks all messages in a conversation as read for a user
func (s *Service) MarkAsRead(conversationID, userID string) error {
	// Verify membership
	isMember, _ := s.repo.IsMember(conversationID, userID)
	if !isMember {
		return errors.New("forbidden")
	}

	err := s.repo.MarkAsRead(conversationID, userID)
	if err != nil {
		return err
	}

	// Publish event so the UI (list) updates across all tabs/devices
	event := events.ConversationReadEvent{
		ConversationID: conversationID,
		UserID:         userID,
		UnreadCount:    0,
	}
	s.producer.PublishConversationRead(event)

	return nil
}

// GetUnreadCount returns the number of unread messages for a user in a conversation
func (s *Service) GetUnreadCount(conversationID, userID string) (int64, error) {
	// Verify membership
	isMember, _ := s.repo.IsMember(conversationID, userID)
	if !isMember {
		return 0, errors.New("forbidden")
	}

	return s.repo.GetUnreadCount(conversationID, userID)
}

// MarkMessageAsRead creates a read receipt for a specific message
func (s *Service) MarkMessageAsRead(conversationID, messageID, userID string) error {
	// Verify membership
	isMember, _ := s.repo.IsMember(conversationID, userID)
	if !isMember {
		return errors.New("forbidden")
	}

	// Verify message belongs to conversation
	msg, err := s.repo.GetMessageByID(messageID)
	if err != nil {
		return errors.New("message not found")
	}
	if msg.ConversationID != conversationID {
		return errors.New("message does not belong to this conversation")
	}

	// Don't create read receipt for own messages
	if msg.SenderID == userID {
		return nil
	}

	return s.repo.CreateReadReceipt(messageID, userID)
}
