package postgres

import (
	"context"
	"errors"

	"github.com/OmarrGhorab/payment-service/internal/domain/cart"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CartRepository struct {
	db *gorm.DB
}

func NewCartRepository(db *gorm.DB) *CartRepository {
	return &CartRepository{db: db}
}

func (r *CartRepository) GetOrCreateCart(ctx context.Context, userID uuid.UUID) (*cart.Cart, error) {
	var c cart.Cart
	err := r.db.WithContext(ctx).
		Preload("Items").
		Where("user_id = ?", userID).
		First(&c).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create new cart
			c = cart.Cart{UserID: userID}
			if err := r.db.WithContext(ctx).Create(&c).Error; err != nil {
				return nil, err
			}
			return &c, nil
		}
		return nil, err
	}

	return &c, nil
}

func (r *CartRepository) AddItem(ctx context.Context, item *cart.CartItem) error {
	// Check if item already exists
	var existing cart.CartItem
	err := r.db.WithContext(ctx).
		Where("cart_id = ? AND course_id = ?", item.CartID, item.CourseID).
		First(&existing).Error

	if err == nil {
		// Update existing item
		return r.db.WithContext(ctx).
			Model(&existing).
			Updates(map[string]interface{}{
				"billing_type": item.BillingType,
				"price_cents":  item.PriceCents,
				"currency":     item.Currency,
			}).Error
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	// Create new item
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *CartRepository) RemoveItem(ctx context.Context, cartID, courseID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("cart_id = ? AND course_id = ?", cartID, courseID).
		Delete(&cart.CartItem{}).Error
}

func (r *CartRepository) ClearCart(ctx context.Context, cartID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("cart_id = ?", cartID).
		Delete(&cart.CartItem{}).Error
}

func (r *CartRepository) GetCartWithItems(ctx context.Context, userID uuid.UUID) (*cart.Cart, error) {
	var c cart.Cart
	err := r.db.WithContext(ctx).
		Preload("Items").
		Where("user_id = ?", userID).
		First(&c).Error

	if err != nil {
		return nil, err
	}

	return &c, nil
}
