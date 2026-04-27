package subscription

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/OmarrGhorab/payment-service/internal/domain/subscription"
	"github.com/OmarrGhorab/payment-service/internal/infrastructure/coursesclient"
	"github.com/OmarrGhorab/payment-service/internal/infrastructure/persistence/postgres"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Service struct {
	repo          *postgres.SubscriptionRepository
	coursesClient *coursesclient.Client
}

func NewService(repo *postgres.SubscriptionRepository, coursesClient *coursesclient.Client) *Service {
	return &Service{
		repo:          repo,
		coursesClient: coursesClient,
	}
}

func (s *Service) CreateSubscription(ctx context.Context, userID, courseID uuid.UUID, priceCents int64, currency string) (*subscription.Subscription, error) {
	// Check if subscription already exists
	existing, err := s.repo.GetByUserAndCourse(ctx, userID, courseID)
	if err == nil && existing.Status == subscription.StatusActive {
		return nil, errors.New("active subscription already exists for this course")
	}

	// Create new subscription
	sub := &subscription.Subscription{
		UserID:          userID,
		CourseID:        courseID,
		Status:          subscription.StatusActive,
		PriceCents:      priceCents,
		Currency:        currency,
		BillingCycle:    subscription.BillingCycleMonthly,
		NextBillingDate: time.Now().AddDate(0, 1, 0), // Next month
		StartedAt:       time.Now(),
	}

	if err := s.repo.Create(ctx, sub); err != nil {
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	return sub, nil
}

func (s *Service) CancelSubscription(ctx context.Context, userID, subscriptionID uuid.UUID) error {
	sub, err := s.repo.GetByID(ctx, subscriptionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("subscription not found")
		}
		return err
	}

	if sub.UserID != userID {
		return errors.New("unauthorized")
	}

	if sub.Status != subscription.StatusActive {
		return errors.New("subscription is not active")
	}

	return s.repo.UpdateStatus(ctx, subscriptionID, subscription.StatusCancelled)
}

func (s *Service) GetUserSubscriptions(ctx context.Context, userID uuid.UUID) ([]subscription.Subscription, error) {
	return s.repo.GetUserSubscriptions(ctx, userID)
}

func (s *Service) GetSubscription(ctx context.Context, userID, subscriptionID uuid.UUID) (*subscription.Subscription, error) {
	sub, err := s.repo.GetByID(ctx, subscriptionID)
	if err != nil {
		return nil, err
	}

	if sub.UserID != userID {
		return nil, errors.New("unauthorized")
	}

	return sub, nil
}

func (s *Service) GetSubscriptionByID(ctx context.Context, subscriptionID uuid.UUID) (*subscription.Subscription, error) {
	return s.repo.GetByID(ctx, subscriptionID)
}

func (s *Service) ProcessDueSubscriptions(ctx context.Context) ([]uuid.UUID, error) {
	// Get all subscriptions due for billing
	dueDate := time.Now()
	subs, err := s.repo.GetDueSubscriptions(ctx, dueDate)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch due subscriptions: %w", err)
	}

	var processedIDs []uuid.UUID

	for _, sub := range subs {
		log.Printf("Processing subscription %s for user %s, course %s", sub.ID, sub.UserID, sub.CourseID)
		
		// Check if enrollment is still active
		isEnrolled, _, err := s.coursesClient.CheckEnrollment(ctx, sub.UserID.String(), sub.CourseID.String())
		if err != nil {
			log.Printf("Error checking enrollment for subscription %s: %v", sub.ID, err)
			continue
		}

		if !isEnrolled {
			log.Printf("User %s no longer enrolled in course %s, suspending subscription", sub.UserID, sub.CourseID)
			s.repo.UpdateStatus(ctx, sub.ID, subscription.StatusSuspended)
			continue
		}

		processedIDs = append(processedIDs, sub.ID)
	}

	return processedIDs, nil
}

func (s *Service) ProcessRenewalReminders(ctx context.Context) ([]uuid.UUID, error) {
	// Look for subscriptions renewing in the next 3 days
	reminderDate := time.Now().AddDate(0, 0, 3)
	subs, err := s.repo.GetUpcomingRenewals(ctx, reminderDate)
	if err != nil {
		return nil, err
	}

	var notifyIDs []uuid.UUID
	for _, sub := range subs {
		// Verify enrollment still active
		isEnrolled, _, err := s.coursesClient.CheckEnrollment(ctx, sub.UserID.String(), sub.CourseID.String())
		if err == nil && isEnrolled {
			notifyIDs = append(notifyIDs, sub.ID)
		}
	}
	return notifyIDs, nil
}

func (s *Service) UpdateBillingDate(ctx context.Context, subscriptionID uuid.UUID, success bool) error {
	sub, err := s.repo.GetByID(ctx, subscriptionID)
	if err != nil {
		return err
	}

	now := time.Now()
	nextBilling := now.AddDate(0, 1, 0) // Next month

	if success {
		return s.repo.UpdateBillingDate(ctx, subscriptionID, now, nextBilling)
	}

	// If payment failed, retry in 3 days
	nextBilling = now.AddDate(0, 0, 3)
	
	// Use current time if last billing date is nil
	lastBilling := now
	if sub.LastBillingDate != nil {
		lastBilling = *sub.LastBillingDate
	}
	
	return s.repo.UpdateBillingDate(ctx, subscriptionID, lastBilling, nextBilling)
}
