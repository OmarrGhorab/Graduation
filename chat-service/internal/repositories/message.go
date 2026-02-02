package repositories

import (
	"context"

	"github.com/graduation/chat-service/internal/models"
	"gorm.io/gorm"
)

// MessageRepository handles database operations for messages
type MessageRepository struct {
	db *gorm.DB
}

// NewMessageRepository creates a new MessageRepository
func NewMessageRepository(db *gorm.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

// Create creates a new message
func (r *MessageRepository) Create(ctx context.Context, message *models.Message) error {
	return r.db.WithContext(ctx).Create(message).Error
}

// GetByID retrieves a message by ID
func (r *MessageRepository) GetByID(ctx context.Context, id string) (*models.Message, error) {
	var message models.Message
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&message).Error
	if err != nil {
		return nil, err
	}
	return &message, nil
}

// GetByIDWithReply retrieves a message with its reply-to message
func (r *MessageRepository) GetByIDWithReply(ctx context.Context, id string) (*models.Message, error) {
	var message models.Message
	err := r.db.WithContext(ctx).
		Preload("ReplyTo").
		Where("id = ?", id).
		First(&message).Error
	if err != nil {
		return nil, err
	}
	return &message, nil
}

// GetConversationMessages retrieves messages for a conversation with pagination
func (r *MessageRepository) GetConversationMessages(ctx context.Context, conversationID string, queryStr string, limit, offset int) ([]models.Message, error) {
	var messages []models.Message
	query := r.db.WithContext(ctx).
		Where("conversation_id = ? AND is_deleted = false", conversationID)

	if queryStr != "" {
		query = query.Where("content ILIKE ?", "%"+queryStr+"%")
	}

	err := query.
		Preload("ReplyTo").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&messages).Error
	return messages, err
}

// GetMessagesAfter retrieves messages after a specific message ID (for long polling)
func (r *MessageRepository) GetMessagesAfter(ctx context.Context, conversationID, afterMessageID string, limit int) ([]models.Message, error) {
	var messages []models.Message

	query := r.db.WithContext(ctx).
		Where("conversation_id = ? AND is_deleted = false", conversationID)

	if afterMessageID != "" {
		// Get the timestamp of the after message
		var afterMsg models.Message
		if err := r.db.WithContext(ctx).Select("created_at").Where("id = ?", afterMessageID).First(&afterMsg).Error; err != nil {
			return nil, err
		}
		query = query.Where("created_at > ?", afterMsg.CreatedAt)
	}

	err := query.
		Preload("ReplyTo").
		Order("created_at ASC").
		Limit(limit).
		Find(&messages).Error

	return messages, err
}

// Update updates a message
func (r *MessageRepository) Update(ctx context.Context, message *models.Message) error {
	return r.db.WithContext(ctx).Save(message).Error
}

// UpdateContent updates the content of a message
func (r *MessageRepository) UpdateContent(ctx context.Context, messageID, content string) error {
	return r.db.WithContext(ctx).
		Model(&models.Message{}).
		Where("id = ?", messageID).
		Updates(map[string]interface{}{
			"content":    content,
			"updated_at": gorm.Expr("NOW()"),
		}).Error
}

// SoftDelete soft deletes a message
func (r *MessageRepository) SoftDelete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).
		Model(&models.Message{}).
		Where("id = ?", id).
		Update("is_deleted", true).Error
}

// Pin creates a pinned message entry
func (r *MessageRepository) Pin(ctx context.Context, pinnedMsg *models.PinnedMessage) error {
	return r.db.WithContext(ctx).Create(pinnedMsg).Error
}

// Unpin removes a pinned message entry
func (r *MessageRepository) Unpin(ctx context.Context, messageID string) error {
	return r.db.WithContext(ctx).Delete(&models.PinnedMessage{}, "message_id = ?", messageID).Error
}

// GetPinnedMessages retrieves all pinned messages for a conversation
func (r *MessageRepository) GetPinnedMessages(ctx context.Context, conversationID string) ([]models.PinnedMessage, error) {
	var pinnedMsgs []models.PinnedMessage
	err := r.db.WithContext(ctx).
		Preload("Message").
		Where("conversation_id = ?", conversationID).
		Order("pinned_at DESC").
		Find(&pinnedMsgs).Error
	return pinnedMsgs, err
}

// IsPinned checks if a message is pinned
func (r *MessageRepository) IsPinned(ctx context.Context, messageID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.PinnedMessage{}).
		Where("message_id = ?", messageID).
		Count(&count).Error
	return count > 0, err
}

// UnpinAllInConversation removes all pinned messages for a conversation
func (r *MessageRepository) UnpinAllInConversation(ctx context.Context, conversationID string) error {
	return r.db.WithContext(ctx).Delete(&models.PinnedMessage{}, "conversation_id = ?", conversationID).Error
}

// GetMediaHistory retrieves messages with media (image, voice) or links for a conversation
func (r *MessageRepository) GetMediaHistory(ctx context.Context, conversationID string, types []models.MessageType, includeLinks bool, limit, offset int) ([]models.Message, error) {
	var messages []models.Message
	query := r.db.WithContext(ctx).
		Where("conversation_id = ? AND is_deleted = false", conversationID)

	if len(types) > 0 && includeLinks {
		// Matches specified types OR contains links (using regex for links)
		query = query.Where("(type IN ? OR content ~* 'https?://[^\\s]+')", types)
	} else if len(types) > 0 {
		query = query.Where("type IN ?", types)
	} else if includeLinks {
		query = query.Where("content ~* 'https?://[^\\s]+'")
	}

	err := query.
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&messages).Error

	return messages, err
}

// GetAllMediaURLsInConversation retrieves all media URLs shared in a conversation
func (r *MessageRepository) GetAllMediaURLsInConversation(ctx context.Context, conversationID string) ([]string, error) {
	var results []struct {
		MediaURLs []string `gorm:"type:text[]"`
	}
	err := r.db.WithContext(ctx).
		Model(&models.Message{}).
		Select("media_urls").
		Where("conversation_id = ? AND media_urls IS NOT NULL AND cardinality(media_urls) > 0", conversationID).
		Find(&results).Error
	if err != nil {
		return nil, err
	}

	var allURLs []string
	for _, res := range results {
		allURLs = append(allURLs, res.MediaURLs...)
	}
	return allURLs, nil
}

// DeleteAllByConversation deletes all messages and pins in a conversation
func (r *MessageRepository) DeleteAllByConversation(ctx context.Context, conversationID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete pins first
		if err := tx.Delete(&models.PinnedMessage{}, "conversation_id = ?", conversationID).Error; err != nil {
			return err
		}
		// Delete messages
		if err := tx.Delete(&models.Message{}, "conversation_id = ?", conversationID).Error; err != nil {
			return err
		}
		return nil
	})
}
