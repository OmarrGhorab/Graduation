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

// ConversationFilter contains filters for querying conversations
type ConversationFilter struct {
	Role  models.UserRole
	Type  models.ConversationType
	Query string
}

// UpdateLastRead updates the last read timestamp for a user in a conversation
func (r *ConversationRepository) UpdateLastRead(ctx context.Context, conversationID, userID string) error {
	return r.db.WithContext(ctx).
		Model(&models.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Update("last_read_at", gorm.Expr("NOW()")).Error
}

// GetUserConversations retrieves all conversations for a user with filters and unread counts
func (r *ConversationRepository) GetUserConversations(ctx context.Context, userID string, filter ConversationFilter, limit, offset int) ([]models.Conversation, error) {
	var conversations []models.Conversation
	query := r.db.WithContext(ctx).
		Table("conversations").
		Select("conversations.*, conversation_members.unread_count, "+
			"(SELECT content FROM messages WHERE messages.conversation_id = conversations.id ORDER BY created_at DESC LIMIT 1) as last_message_content, "+
			"(SELECT sender_id FROM messages WHERE messages.conversation_id = conversations.id ORDER BY created_at DESC LIMIT 1) as last_message_sender_id").
		Joins("JOIN conversation_members ON conversation_members.conversation_id = conversations.id").
		Where("conversation_members.user_id = ? AND conversation_members.left_at IS NULL", userID)

	if filter.Role != "" {
		query = query.Where("conversation_members.user_role = ?", filter.Role)
	}

	if filter.Type != "" {
		query = query.Where("conversations.type = ?", filter.Type)
	}

	if filter.Query != "" {
		query = query.Where("conversations.name ILIKE ?", "%"+filter.Query+"%")
	}

	err := query.
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

// UpdateImage updates the conversation's profile image
func (r *ConversationRepository) UpdateImage(ctx context.Context, id string, imageURL string) error {
	return r.db.WithContext(ctx).
		Model(&models.Conversation{}).
		Where("id = ?", id).
		Update("image_url", imageURL).Error
}

// Delete deletes a conversation
func (r *ConversationRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&models.Conversation{}, "id = ?", id).Error
}
