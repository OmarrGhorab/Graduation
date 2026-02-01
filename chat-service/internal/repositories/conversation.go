package repositories

import (
	"context"

	"github.com/graduation/chat-service/internal/models"
	"gorm.io/gorm"
)

// ConversationRepository handles database operations for conversations
type ConversationRepository struct {
	db *gorm.DB
}

// NewConversationRepository creates a new ConversationRepository
func NewConversationRepository(db *gorm.DB) *ConversationRepository {
	return &ConversationRepository{db: db}
}

// Create creates a new conversation
func (r *ConversationRepository) Create(ctx context.Context, conversation *models.Conversation) error {
	return r.db.WithContext(ctx).Create(conversation).Error
}

// GetByID retrieves a conversation by ID
func (r *ConversationRepository) GetByID(ctx context.Context, id string) (*models.Conversation, error) {
	var conversation models.Conversation
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&conversation).Error
	if err != nil {
		return nil, err
	}
	return &conversation, nil
}

// GetByIDWithMembers retrieves a conversation with its members
func (r *ConversationRepository) GetByIDWithMembers(ctx context.Context, id string) (*models.Conversation, error) {
	var conversation models.Conversation
	err := r.db.WithContext(ctx).
		Preload("Members", "left_at IS NULL").
		Where("id = ?", id).
		First(&conversation).Error
	if err != nil {
		return nil, err
	}
	return &conversation, nil
}

// GetUserConversations retrieves all conversations for a user
func (r *ConversationRepository) GetUserConversations(ctx context.Context, userID string, limit, offset int) ([]models.Conversation, error) {
	var conversations []models.Conversation
	err := r.db.WithContext(ctx).
		Joins("JOIN conversation_members ON conversation_members.conversation_id = conversations.id").
		Where("conversation_members.user_id = ? AND conversation_members.left_at IS NULL", userID).
		Preload("Members", "left_at IS NULL").
		Order("conversations.updated_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&conversations).Error
	return conversations, err
}

// FindDirectChat finds an existing direct chat between two users
func (r *ConversationRepository) FindDirectChat(ctx context.Context, userID1, userID2 string) (*models.Conversation, error) {
	var conversation models.Conversation
	err := r.db.WithContext(ctx).
		Joins("JOIN conversation_members cm1 ON cm1.conversation_id = conversations.id AND cm1.user_id = ? AND cm1.left_at IS NULL", userID1).
		Joins("JOIN conversation_members cm2 ON cm2.conversation_id = conversations.id AND cm2.user_id = ? AND cm2.left_at IS NULL", userID2).
		Where("conversations.type = ?", models.ConversationTypeDirect).
		First(&conversation).Error
	if err != nil {
		return nil, err
	}
	return &conversation, nil
}

// Update updates a conversation
func (r *ConversationRepository) Update(ctx context.Context, conversation *models.Conversation) error {
	return r.db.WithContext(ctx).Save(conversation).Error
}

// Delete deletes a conversation
func (r *ConversationRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&models.Conversation{}, "id = ?", id).Error
}
