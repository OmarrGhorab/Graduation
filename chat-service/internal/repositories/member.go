package repositories

import (
	"context"

	"github.com/graduation/chat-service/internal/models"
	"gorm.io/gorm"
)

// MemberRepository handles database operations for conversation members
type MemberRepository struct {
	db *gorm.DB
}

// NewMemberRepository creates a new MemberRepository
func NewMemberRepository(db *gorm.DB) *MemberRepository {
	return &MemberRepository{db: db}
}

// Create adds a member to a conversation
func (r *MemberRepository) Create(ctx context.Context, member *models.ConversationMember) error {
	return r.db.WithContext(ctx).Create(member).Error
}

// GetByConversationAndUser retrieves a member by conversation and user ID
func (r *MemberRepository) GetByConversationAndUser(ctx context.Context, conversationID, userID string) (*models.ConversationMember, error) {
	var member models.ConversationMember
	err := r.db.WithContext(ctx).
		Where("conversation_id = ? AND user_id = ? AND left_at IS NULL", conversationID, userID).
		First(&member).Error
	if err != nil {
		return nil, err
	}
	return &member, nil
}

// GetConversationMembers retrieves all active members of a conversation
func (r *MemberRepository) GetConversationMembers(ctx context.Context, conversationID string) ([]models.ConversationMember, error) {
	var members []models.ConversationMember
	err := r.db.WithContext(ctx).
		Where("conversation_id = ? AND left_at IS NULL", conversationID).
		Find(&members).Error
	return members, err
}

// GetConversationMemberIDs retrieves all active member user IDs of a conversation
func (r *MemberRepository) GetConversationMemberIDs(ctx context.Context, conversationID string) ([]string, error) {
	var userIDs []string
	err := r.db.WithContext(ctx).
		Model(&models.ConversationMember{}).
		Where("conversation_id = ? AND left_at IS NULL", conversationID).
		Pluck("user_id", &userIDs).Error
	return userIDs, err
}

// Update updates a member
func (r *MemberRepository) Update(ctx context.Context, member *models.ConversationMember) error {
	return r.db.WithContext(ctx).Save(member).Error
}

// UpdateRole updates a member's role
func (r *MemberRepository) UpdateRole(ctx context.Context, conversationID, userID string, memberRole models.MemberRole) error {
	return r.db.WithContext(ctx).
		Model(&models.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ? AND left_at IS NULL", conversationID, userID).
		Update("member_role", memberRole).Error
}

// Remove soft-removes a member by setting left_at
func (r *MemberRepository) Remove(ctx context.Context, conversationID, userID string) error {
	return r.db.WithContext(ctx).
		Model(&models.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ? AND left_at IS NULL", conversationID, userID).
		Update("left_at", gorm.Expr("NOW()")).Error
}

// IsMember checks if a user is an active member of a conversation
func (r *MemberRepository) IsMember(ctx context.Context, conversationID, userID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ? AND left_at IS NULL", conversationID, userID).
		Count(&count).Error
	return count > 0, err
}

// CountMembers counts active members in a conversation
func (r *MemberRepository) CountMembers(ctx context.Context, conversationID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.ConversationMember{}).
		Where("conversation_id = ? AND left_at IS NULL", conversationID).
		Count(&count).Error
	return count, err
}

// IncrementUnreadCount increments unread count for all members except the excluded ones
func (r *MemberRepository) IncrementUnreadCount(ctx context.Context, conversationID string, excludeUserIDs []string) error {
	query := r.db.WithContext(ctx).
		Model(&models.ConversationMember{}).
		Where("conversation_id = ? AND left_at IS NULL", conversationID)

	if len(excludeUserIDs) > 0 {
		query = query.Where("user_id NOT IN ?", excludeUserIDs)
	}

	return query.UpdateColumn("unread_count", gorm.Expr("unread_count + ?", 1)).Error
}

// ResetUnreadCount resets unread count for a user
func (r *MemberRepository) ResetUnreadCount(ctx context.Context, conversationID, userID string, lastReadMessageID *string) error {
	updates := map[string]interface{}{
		"unread_count": 0,
		"last_read_at": gorm.Expr("NOW()"),
	}

	if lastReadMessageID != nil {
		updates["last_read_message_id"] = *lastReadMessageID
	}

	return r.db.WithContext(ctx).
		Model(&models.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Updates(updates).Error
}

// DeleteAllByConversation deletes all members in a conversation
func (r *MemberRepository) DeleteAllByConversation(ctx context.Context, conversationID string) error {
	return r.db.WithContext(ctx).Delete(&models.ConversationMember{}, "conversation_id = ?", conversationID).Error
}
