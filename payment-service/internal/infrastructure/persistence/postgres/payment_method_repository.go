package postgres

import (
	"context"
	"time"

	"github.com/OmarrGhorab/payment-service/internal/domain/paymentmethod"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PaymentMethodRepository struct {
	db *gorm.DB
}

func NewPaymentMethodRepository(db *gorm.DB) *PaymentMethodRepository {
	return &PaymentMethodRepository{db: db}
}

func (r *PaymentMethodRepository) Create(ctx context.Context, pm *paymentmethod.PaymentMethod) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// If this is set as default, unset other defaults
		if pm.IsDefault {
			if err := tx.Model(&paymentmethod.PaymentMethod{}).
				Where("user_id = ? AND is_default = ?", pm.UserID, true).
				Update("is_default", false).Error; err != nil {
				return err
			}
		}

		return tx.Create(pm).Error
	})
}

func (r *PaymentMethodRepository) GetByID(ctx context.Context, id uuid.UUID) (*paymentmethod.PaymentMethod, error) {
	var pm paymentmethod.PaymentMethod
	err := r.db.WithContext(ctx).Where("id = ? AND is_active = ?", id, true).First(&pm).Error
	if err != nil {
		return nil, err
	}
	return &pm, nil
}

func (r *PaymentMethodRepository) GetUserPaymentMethods(ctx context.Context, userID uuid.UUID) ([]paymentmethod.PaymentMethod, error) {
	var methods []paymentmethod.PaymentMethod
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND is_active = ?", userID, true).
		Order("is_default DESC, created_at DESC").
		Find(&methods).Error
	return methods, err
}

func (r *PaymentMethodRepository) GetDefaultPaymentMethod(ctx context.Context, userID uuid.UUID) (*paymentmethod.PaymentMethod, error) {
	var pm paymentmethod.PaymentMethod
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND is_default = ? AND is_active = ?", userID, true, true).
		First(&pm).Error
	if err != nil {
		return nil, err
	}
	return &pm, nil
}

func (r *PaymentMethodRepository) SetDefault(ctx context.Context, userID, paymentMethodID uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Unset all defaults for this user
		if err := tx.Model(&paymentmethod.PaymentMethod{}).
			Where("user_id = ?", userID).
			Update("is_default", false).Error; err != nil {
			return err
		}

		// Set the new default
		return tx.Model(&paymentmethod.PaymentMethod{}).
			Where("id = ? AND user_id = ?", paymentMethodID, userID).
			Updates(map[string]interface{}{
				"is_default": true,
				"updated_at": time.Now(),
			}).Error
	})
}

func (r *PaymentMethodRepository) Delete(ctx context.Context, userID, paymentMethodID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&paymentmethod.PaymentMethod{}).
		Where("id = ? AND user_id = ?", paymentMethodID, userID).
		Updates(map[string]interface{}{
			"is_active":  false,
			"is_default": false,
			"updated_at": time.Now(),
		}).Error
}
