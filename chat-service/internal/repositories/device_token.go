package repositories

import (
	"context"

	"github.com/graduation/chat-service/internal/models"
	"gorm.io/gorm"
)

// DeviceTokenRepository handles database operations for device tokens
type DeviceTokenRepository struct {
	db *gorm.DB
}

// NewDeviceTokenRepository creates a new DeviceTokenRepository
func NewDeviceTokenRepository(db *gorm.DB) *DeviceTokenRepository {
	return &DeviceTokenRepository{db: db}
}

// Create creates or updates a device token
func (r *DeviceTokenRepository) Create(ctx context.Context, token *models.DeviceToken) error {
	return r.db.WithContext(ctx).
		Clauses().
		Create(token).Error
}

// GetUserTokens retrieves all active tokens for a user
func (r *DeviceTokenRepository) GetUserTokens(ctx context.Context, userID string) ([]models.DeviceToken, error) {
	var tokens []models.DeviceToken
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND is_active = true", userID).
		Find(&tokens).Error
	return tokens, err
}

// GetUserTokensByPlatform retrieves active tokens for a user filtered by platform
func (r *DeviceTokenRepository) GetUserTokensByPlatform(ctx context.Context, userID, platform string) ([]models.DeviceToken, error) {
	var tokens []models.DeviceToken
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND platform = ? AND is_active = true", userID, platform).
		Find(&tokens).Error
	return tokens, err
}

// Deactivate deactivates a device token
func (r *DeviceTokenRepository) Deactivate(ctx context.Context, token string) error {
	return r.db.WithContext(ctx).
		Model(&models.DeviceToken{}).
		Where("token = ?", token).
		Update("is_active", false).Error
}

// DeactivateAllForUser deactivates all tokens for a user
func (r *DeviceTokenRepository) DeactivateAllForUser(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).
		Model(&models.DeviceToken{}).
		Where("user_id = ?", userID).
		Update("is_active", false).Error
}

// Delete removes a device token
func (r *DeviceTokenRepository) Delete(ctx context.Context, token string) error {
	return r.db.WithContext(ctx).
		Delete(&models.DeviceToken{}, "token = ?", token).Error
}
