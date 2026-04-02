package postgres

import (
	"context"
	"time"

	"github.com/OmarrGhorab/payment-service/internal/domain/subscription"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SubscriptionRepository struct {
	db *gorm.DB
}

func NewSubscriptionRepository(db *gorm.DB) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

func (r *SubscriptionRepository) Create(ctx context.Context, sub *subscription.Subscription) error {
	return r.db.WithContext(ctx).Create(sub).Error
}

func (r *SubscriptionRepository) GetByID(ctx context.Context, id uuid.UUID) (*subscription.Subscription, error) {
	var sub subscription.Subscription
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&sub).Error
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

func (r *SubscriptionRepository) GetByUserAndCourse(ctx context.Context, userID, courseID uuid.UUID) (*subscription.Subscription, error) {
	var sub subscription.Subscription
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND course_id = ?", userID, courseID).
		First(&sub).Error
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

func (r *SubscriptionRepository) GetUserSubscriptions(ctx context.Context, userID uuid.UUID) ([]subscription.Subscription, error) {
	var subs []subscription.Subscription
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&subs).Error
	return subs, err
}

func (r *SubscriptionRepository) GetDueSubscriptions(ctx context.Context, beforeDate time.Time) ([]subscription.Subscription, error) {
	var subs []subscription.Subscription
	err := r.db.WithContext(ctx).
		Where("status = ? AND next_billing_date <= ?", subscription.StatusActive, beforeDate).
		Find(&subs).Error
	return subs, err
}

func (r *SubscriptionRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status subscription.SubscriptionStatus) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}

	if status == subscription.StatusCancelled {
		updates["cancelled_at"] = time.Now()
	}

	return r.db.WithContext(ctx).
		Model(&subscription.Subscription{}).
		Where("id = ?", id).
		Updates(updates).Error
}

func (r *SubscriptionRepository) UpdateBillingDate(ctx context.Context, id uuid.UUID, lastBilling, nextBilling time.Time) error {
	return r.db.WithContext(ctx).
		Model(&subscription.Subscription{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"last_billing_date": lastBilling,
			"next_billing_date": nextBilling,
			"updated_at":        time.Now(),
		}).Error
}
