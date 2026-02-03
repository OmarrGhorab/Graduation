package repository

import (
	"github.com/graduation/chat-service/internal/models"
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
	return r.db.Create(m).Error
}

func (r *Repository) GetMessages(conversationID string, limit, offset int) ([]models.Message, error) {
	var messages []models.Message
	err := r.db.Where("conversation_id = ?", conversationID).
		Order("created_at desc").
		Find(&messages).Error
	return messages, err
}

func (r *Repository) PinMessage(pm *models.PinnedMessage) error {
	// 1. Unpin any existing pins (if we enforce 1 pin per conv, or just let it exist)
	// Requirement: "same functionality" often implies single pin or list.
	// Original code (Step 24) showed "UnpinAllInConversation".
	// So we enforce single pin.
	r.db.Where("conversation_id = ?", pm.ConversationID).Delete(&models.PinnedMessage{})
	return r.db.Create(pm).Error
}

func (r *Repository) UnpinMessage(conversationID, messageID string) error {
	return r.db.Where("conversation_id = ? AND message_id = ?", conversationID, messageID).
		Delete(&models.PinnedMessage{}).Error
}

func (r *Repository) GetPinnedMessages(conversationID string) ([]models.PinnedMessage, error) {
	var pins []models.PinnedMessage
	err := r.db.Preload("Message").Where("conversation_id = ?", conversationID).Find(&pins).Error
	return pins, err
}
