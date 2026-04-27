package repository

import (
	"fmt"
	"sort"
	"time"

	"github.com/graduation/chat-service/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Ensure Transaction Support
func (r *Repository) WithTx(tx *gorm.DB) *Repository {
	return &Repository{db: tx}
}

// --- Conversations ---

func (r *Repository) CreateConversation(c *models.Conversation) error {
	return r.db.Create(c).Error
}

func (r *Repository) GetConversationByID(id string) (*models.Conversation, error) {
	var c models.Conversation
	if err := r.db.Preload("Members").First(&c, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *Repository) GetUserConversations(userID string) ([]models.Conversation, error) {
	// Find conversation IDs where user is a member
	var memberEntries []models.ConversationMember
	if err := r.db.Where("user_id = ?", userID).Find(&memberEntries).Error; err != nil {
		return nil, err
	}

	if len(memberEntries) == 0 {
		return []models.Conversation{}, nil
	}

	ids := make([]string, len(memberEntries))
	for i, m := range memberEntries {
		ids[i] = m.ConversationID
	}

	var conversations []models.Conversation
	// Preload members might be heavy for list, maybe just fetch basic info
	if err := r.db.Where("id IN ?", ids).Order("updated_at desc").Find(&conversations).Error; err != nil {
		return nil, err
	}
	return conversations, nil
}

func (r *Repository) FindOrCreateDirectConversation(user1, user2 string) (*models.Conversation, error) {
	uids := []string{user1, user2}
	sort.Strings(uids)
	// Use MD5 hash of the name to create a deterministic UUID that is actually a valid UUID!
	// This ensures it fits into the DB's UUID column.
	source := fmt.Sprintf("direct:%s:%s", uids[0], uids[1])
	convID := uuid.NewMD5(uuid.NameSpaceDNS, []byte(source)).String()

	// 2. Try to find by this deterministic ID first!
	var existing models.Conversation
	if err := r.db.Preload("Members").First(&existing, "id = ?", convID).Error; err == nil {
		return &existing, nil
	}

	// 3. Not found, create a new one
	conv := &models.Conversation{
		ID:        convID,
		Type:      models.Direct,
		CreatedBy: user1,
		Members: []models.ConversationMember{
			{UserID: user1, Role: models.RoleMember, JoinedAt: time.Now()},
			{UserID: user2, Role: models.RoleMember, JoinedAt: time.Now()},
		},
		UpdatedAt: time.Now(),
	}

	if err := r.db.Create(conv).Error; err != nil {
		// If someone else created it in the meantime, retry fetching
		if err == gorm.ErrDuplicatedKey {
			if err := r.db.Preload("Members").First(&existing, "id = ?", convID).Error; err == nil {
				return &existing, nil
			}
		}
		return nil, err
	}
	return conv, nil
}

func (r *Repository) UpsertCourseGroup(courseID, name, creatorID, imageURL string) (*models.Conversation, error) {
	var conv models.Conversation
	if err := r.db.First(&conv, "id = ?", courseID).Error; err == nil {
		// Update name and image if exists without touching updated_at
		return &conv, r.db.Model(&conv).UpdateColumns(map[string]interface{}{
			"name":      name,
			"image_url": imageURL,
		}).Error
	}

	newConv := &models.Conversation{
		ID:          courseID,
		Type:        models.Group,
		Name:        name,
		Description: "Official Course Group",
		ImageURL:    imageURL,
		CreatedBy:   creatorID,
		UpdatedAt:   time.Now(),
	}

	if err := r.db.Create(newConv).Error; err != nil {
		if err == gorm.ErrDuplicatedKey {
			if err := r.db.First(&conv, "id = ?", courseID).Error; err == nil {
				return &conv, nil
			}
		}
		return nil, err
	}
	return newConv, nil
}

func (r *Repository) EnsureMembership(conversationID, userID string) error {
	var m models.ConversationMember
	err := r.db.Where("conversation_id = ? AND user_id = ?", conversationID, userID).First(&m).Error
	if err == nil {
		return nil // already a member
	}

	m = models.ConversationMember{
		ConversationID: conversationID,
		UserID:         userID,
		Role:           models.RoleMember,
		JoinedAt:       time.Now(),
	}
	return r.db.Create(&m).Error
}

// --- Members ---

func (r *Repository) AddMember(m *models.ConversationMember) error {
	return r.db.Create(m).Error
}

func (r *Repository) IsMember(conversationID, userID string) (bool, error) {
	var count int64
	err := r.db.Model(&models.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Count(&count).Error
	return count > 0, err
}

func (r *Repository) GetMembers(conversationID string) ([]models.ConversationMember, error) {
	var members []models.ConversationMember
	err := r.db.Where("conversation_id = ?", conversationID).Find(&members).Error
	return members, err
}

func (r *Repository) GetMember(conversationID, userID string) (*models.ConversationMember, error) {
	var member models.ConversationMember
	err := r.db.Where("conversation_id = ? AND user_id = ?", conversationID, userID).First(&member).Error
	if err != nil {
		return nil, err
	}
	return &member, nil
}

func (r *Repository) RemoveMember(conversationID, userID string) error {
	return r.db.Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Delete(&models.ConversationMember{}).Error
}

func (r *Repository) UpdateMemberRole(conversationID, userID string, role models.MemberRole) error {
	return r.db.Model(&models.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Update("role", role).Error
}

func (r *Repository) DeleteConversation(conversationID string) error {
	// Delete conversation (cascade should handle members and messages if configured)
	// Otherwise, manually delete related records
	tx := r.db.Begin()
	
	// Delete members
	if err := tx.Where("conversation_id = ?", conversationID).Delete(&models.ConversationMember{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	
	// Delete messages
	if err := tx.Where("conversation_id = ?", conversationID).Delete(&models.Message{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	
	// Delete pinned messages
	if err := tx.Where("conversation_id = ?", conversationID).Delete(&models.PinnedMessage{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	
	// Delete conversation
	if err := tx.Where("id = ?", conversationID).Delete(&models.Conversation{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	
	return tx.Commit().Error
}

// --- Messages ---

func (r *Repository) CreateMessage(m *models.Message) error {
	// Start a transaction to create message and update conversation
	tx := r.db.Begin()
	
	// Create the message
	if err := tx.Create(m).Error; err != nil {
		tx.Rollback()
		return err
	}
	
	// Update the conversation's updated_at timestamp
	if err := tx.Model(&models.Conversation{}).
		Where("id = ?", m.ConversationID).
		Update("updated_at", time.Now()).Error; err != nil {
		tx.Rollback()
		return err
	}
	
	return tx.Commit().Error
}

func (r *Repository) GetMessages(conversationID string, search string, limit, offset int) ([]models.Message, error) {
	var messages []models.Message
	query := r.db.Where("conversation_id = ? AND is_deleted = ?", conversationID, false)
	
	// Apply search filter if provided
	if search != "" {
		query = query.Where("content ILIKE ?", "%"+search+"%")
	}
	
	err := query.Order("created_at desc").
		Limit(limit).
		Offset(offset).
		Find(&messages).Error
	return messages, err
}

func (r *Repository) GetMessageByID(messageID string) (*models.Message, error) {
	var message models.Message
	err := r.db.Where("id = ?", messageID).First(&message).Error
	if err != nil {
		return nil, err
	}
	return &message, nil
}

func (r *Repository) DeleteMessage(messageID string) error {
	// Soft delete - set is_deleted to true
	return r.db.Model(&models.Message{}).
		Where("id = ?", messageID).
		Update("is_deleted", true).Error
}

func (r *Repository) PinMessage(pm *models.PinnedMessage) error {
	// Allow multiple pinned messages per conversation
	return r.db.Create(pm).Error
}

func (r *Repository) UnpinMessage(conversationID, messageID string) error {
	return r.db.Where("conversation_id = ? AND message_id = ?", conversationID, messageID).
		Delete(&models.PinnedMessage{}).Error
}

func (r *Repository) GetPinnedMessages(conversationID string) ([]models.PinnedMessage, error) {
	var pins []models.PinnedMessage
	err := r.db.Preload("Message").Where("conversation_id = ?", conversationID).Order("pinned_at desc").Find(&pins).Error
	return pins, err
}

// Get messages by type for media collection
func (r *Repository) GetMessagesByType(conversationID string, messageType models.MessageType) ([]models.Message, error) {
	var messages []models.Message
	err := r.db.Where("conversation_id = ? AND type = ? AND is_deleted = ?", conversationID, messageType, false).
		Order("created_at desc").
		Find(&messages).Error
	return messages, err
}

// Get messages with links (text messages containing URLs)
func (r *Repository) GetMessagesWithLinks(conversationID string) ([]models.Message, error) {
	var messages []models.Message
	// Match messages containing http:// or https://
	err := r.db.Where("conversation_id = ? AND type = ? AND is_deleted = ? AND (content LIKE ? OR content LIKE ?)", 
		conversationID, models.Text, false, "%http://%", "%https://%").
		Order("created_at desc").
		Find(&messages).Error
	return messages, err
}

// --- Read Receipts ---

// MarkAsRead updates the last_read_at timestamp for a user in a conversation
func (r *Repository) MarkAsRead(conversationID, userID string) error {
	return r.db.Model(&models.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Update("last_read_at", time.Now()).Error
}

// GetUnreadCount returns the number of unread messages for a user in a conversation
func (r *Repository) GetUnreadCount(conversationID, userID string) (int64, error) {
	var member models.ConversationMember
	if err := r.db.Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		First(&member).Error; err != nil {
		return 0, err
	}

	var count int64
	err := r.db.Model(&models.Message{}).
		Where("conversation_id = ? AND sender_id != ? AND created_at > ? AND is_deleted = ?",
			conversationID, userID, member.LastReadAt, false).
		Count(&count).Error
	
	return count, err
}

// CreateReadReceipt creates a read receipt for a message
func (r *Repository) CreateReadReceipt(messageID, userID string) error {
	receipt := &models.ReadReceipt{
		MessageID: messageID,
		UserID:    userID,
	}
	// Use FirstOrCreate to avoid duplicates
	return r.db.Where("message_id = ? AND user_id = ?", messageID, userID).
		FirstOrCreate(receipt).Error
}

// GetReadReceipts returns all read receipts for a message
func (r *Repository) GetReadReceipts(messageID string) ([]models.ReadReceipt, error) {
	var receipts []models.ReadReceipt
	err := r.db.Where("message_id = ?", messageID).Find(&receipts).Error
	return receipts, err
}

// GetReadReceiptsForMessages returns read receipts for multiple messages
func (r *Repository) GetReadReceiptsForMessages(messageIDs []string) (map[string][]models.ReadReceipt, error) {
	var receipts []models.ReadReceipt
	err := r.db.Where("message_id IN ?", messageIDs).Find(&receipts).Error
	if err != nil {
		return nil, err
	}

	// Group by message ID
	result := make(map[string][]models.ReadReceipt)
	for _, receipt := range receipts {
		result[receipt.MessageID] = append(result[receipt.MessageID], receipt)
	}
	
	return result, nil
}
