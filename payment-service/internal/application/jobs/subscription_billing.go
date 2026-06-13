package jobs

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/OmarrGhorab/payment-service/internal/application/payment"
	subscriptionApp "github.com/OmarrGhorab/payment-service/internal/application/subscription"
	subscriptionDomain "github.com/OmarrGhorab/payment-service/internal/domain/subscription"
	"github.com/OmarrGhorab/payment-service/internal/infrastructure/authclient"
	"github.com/OmarrGhorab/payment-service/internal/infrastructure/coursesclient"
	"github.com/OmarrGhorab/payment-service/internal/infrastructure/messaging/kafka"
	"github.com/OmarrGhorab/payment-service/internal/infrastructure/notification"
	"github.com/OmarrGhorab/payment-service/internal/infrastructure/paymob"
	"github.com/OmarrGhorab/payment-service/internal/infrastructure/persistence/postgres"
	"github.com/google/uuid"
)

type SubscriptionBillingJob struct {
	subscriptionService *subscriptionApp.Service
	paymentService      *payment.Service
	subscriptionRepo    *postgres.SubscriptionRepository
	paymentMethodRepo   *postgres.PaymentMethodRepository
	emailService        *notification.EmailService
	kafkaProducer       *kafka.Producer
	authClient          *authclient.Client
	coursesClient       *coursesclient.Client
}

func NewSubscriptionBillingJob(
	subSvc *subscriptionApp.Service,
	paySvc *payment.Service,
	subRepo *postgres.SubscriptionRepository,
	pmRepo *postgres.PaymentMethodRepository,
	emailSvc *notification.EmailService,
	kafkaProducer *kafka.Producer,
	authClient *authclient.Client,
	coursesClient *coursesclient.Client,
) *SubscriptionBillingJob {
	return &SubscriptionBillingJob{
		subscriptionService: subSvc,
		paymentService:      paySvc,
		subscriptionRepo:    subRepo,
		paymentMethodRepo:   pmRepo,
		emailService:        emailSvc,
		kafkaProducer:       kafkaProducer,
		authClient:          authClient,
		coursesClient:       coursesClient,
	}
}

// Run processes all due subscriptions
func (j *SubscriptionBillingJob) Run(ctx context.Context) error {
	log.Println("Starting subscription billing job...")

	dueSubscriptions, err := j.subscriptionService.ProcessDueSubscriptions(ctx)
	if err != nil {
		log.Printf("Error fetching due subscriptions: %v", err)
		return err
	}

	if len(dueSubscriptions) > 0 {
		log.Printf("Found %d subscriptions due for billing", len(dueSubscriptions))
		for _, subID := range dueSubscriptions {
			if err := j.processSubscription(ctx, subID); err != nil {
				log.Printf("Error processing subscription %s: %v", subID, err)
				continue
			}
		}
	} else {
		log.Println("No subscriptions due for billing")
	}

	// 2. Process pending reminders
	reminderIDs, err := j.subscriptionService.ProcessRenewalReminders(ctx)
	if err == nil && len(reminderIDs) > 0 {
		log.Printf("Found %d subscriptions needing renewal reminders", len(reminderIDs))
		for _, subID := range reminderIDs {
			if err := j.processReminder(ctx, subID); err != nil {
				log.Printf("Error sending reminder for sub %s: %v", subID, err)
			}
		}
	}

	log.Println("Subscription billing job completed")
	return nil
}

func (j *SubscriptionBillingJob) processReminder(ctx context.Context, subscriptionID uuid.UUID) error {
	sub, err := j.subscriptionRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return err
	}

	amount := fmt.Sprintf("%.2f", float64(sub.PriceCents)/100)

	// Fetch real user name and email
	userName := "Student"
	userEmail := ""
	if userInfo, err := j.authClient.GetUserInfo(ctx, sub.UserID.String()); err == nil && userInfo != nil {
		userName = userInfo.Name
		userEmail = userInfo.Email
	} else {
		log.Printf("[BillingJob] Failed to fetch user info for %s: %v", sub.UserID, err)
	}

	// Fetch real course name
	courseName := "your course"
	if courseInfo, err := j.coursesClient.GetCourseByID(ctx, sub.CourseID.String()); err == nil && courseInfo != nil {
		courseName = courseInfo.Title
	} else {
		log.Printf("[BillingJob] Failed to fetch course info for %s: %v", sub.CourseID, err)
	}

	daysLeft := int(time.Until(sub.NextBillingDate).Hours() / 24)
	if daysLeft < 1 {
		daysLeft = 1
	}

	// 1. Send Email Reminder to student
	if userEmail != "" {
		emailData := notification.SubscriptionRenewalEmail{
			UserName:        userName,
			CourseName:      courseName,
			Amount:          amount,
			Currency:        sub.Currency,
			NextBillingDate: sub.NextBillingDate.Format("January 2, 2006"),
			SubscriptionID:  sub.ID.String(),
		}
		if err := j.emailService.SendSubscriptionRenewalNotification(ctx, userEmail, emailData); err != nil {
			log.Printf("[BillingJob] Failed to email reminder for sub %s: %v", sub.ID, err)
		}
	}

	// 2. Emit Kafka notification for the student
	studentEvent := map[string]interface{}{
		"event_type":      "subscription.renewal_soon",
		"user_id":         sub.UserID.String(),
		"subscription_id": sub.ID.String(),
		"course_id":       sub.CourseID.String(),
		"course_name":     courseName,
		"amount":          amount,
		"currency":        sub.Currency,
		"next_billing":    sub.NextBillingDate.Format("2006-01-02"),
		"days_left":       daysLeft,
	}
	if err := j.kafkaProducer.Publish(ctx, "notifications.v1", sub.UserID.String(), studentEvent); err != nil {
		log.Printf("[BillingJob] Failed to publish student renewal event: %v", err)
	}

	// 3. Emit Kafka notification for each parent of the student
	parents, err := j.authClient.GetParents(ctx, sub.UserID.String())
	if err != nil {
		log.Printf("[BillingJob] Failed to fetch parents for user %s: %v", sub.UserID, err)
	}
	for _, parent := range parents {
		parentEvent := map[string]interface{}{
			"event_type":      "subscription.renewal_soon",
			"user_id":         parent.ID,
			"subscription_id": sub.ID.String(),
			"course_id":       sub.CourseID.String(),
			"course_name":     courseName,
			"child_id":        sub.UserID.String(),
			"child_name":      userName,
			"amount":          amount,
			"currency":        sub.Currency,
			"next_billing":    sub.NextBillingDate.Format("2006-01-02"),
			"days_left":       daysLeft,
		}
		if err := j.kafkaProducer.Publish(ctx, "notifications.v1", parent.ID, parentEvent); err != nil {
			log.Printf("[BillingJob] Failed to publish parent renewal event for parent %s: %v", parent.ID, err)
		}
	}

	// 4. Mark reminder as sent
	return j.subscriptionRepo.UpdateReminderSent(ctx, sub.ID)
}

func (j *SubscriptionBillingJob) processSubscription(ctx context.Context, subscriptionID uuid.UUID) error {
	log.Printf("Processing subscription %s", subscriptionID)

	sub, err := j.subscriptionRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	// Check if subscription has a stored payment method
	if sub.PaymentMethodID != nil {
		log.Printf("Attempting automatic charge for subscription %s using payment method %s", subscriptionID, *sub.PaymentMethodID)
		
		_, err := j.paymentMethodRepo.GetByID(ctx, *sub.PaymentMethodID)
		if err != nil {
			log.Printf("Failed to get payment method %s: %v", *sub.PaymentMethodID, err)
			return j.createManualRenewalPayment(ctx, sub)
		}

		// TODO: Implement automatic charging with Paymob's tokenization API
		log.Printf("Automatic charging not yet implemented. Creating manual payment for subscription %s", subscriptionID)
		return j.createManualRenewalPayment(ctx, sub)
	}

	return j.createManualRenewalPayment(ctx, sub)
}

func (j *SubscriptionBillingJob) createManualRenewalPayment(ctx context.Context, sub *subscriptionDomain.Subscription) error {
	paymentURL, orderID, err := j.paymentService.RenewSubscription(ctx, payment.RenewSubscriptionOptions{
		SubscriptionID: sub.ID,
		BillingData: paymob.BillingData{
			FirstName:   "Subscription",
			LastName:    "Renewal",
			Email:       "renewal@example.com",
			PhoneNumber: "01000000000",
			Street:      "N/A",
			Building:    "N/A",
			Floor:       "N/A",
			Apartment:   "N/A",
			City:        "N/A",
			State:       "N/A",
			Country:     "Egypt",
		},
	})

	if err != nil {
		return fmt.Errorf("failed to create renewal payment: %w", err)
	}

	log.Printf("Created renewal payment for subscription %s: order %s", sub.ID, orderID)

	// Send email notification
	amount := fmt.Sprintf("%.2f", float64(sub.PriceCents)/100)
	emailData := notification.SubscriptionRenewalEmail{
		UserName:        "Valued Customer",
		CourseName:      "Your Course",
		Amount:          amount,
		Currency:        sub.Currency,
		NextBillingDate: sub.NextBillingDate.Format("January 2, 2006"),
		PaymentURL:      paymentURL,
		SubscriptionID:  sub.ID.String(),
	}

	userEmail := "user@example.com" // TODO: Get from auth service
	
	if err := j.emailService.SendSubscriptionRenewalNotification(ctx, userEmail, emailData); err != nil {
		log.Printf("Failed to send renewal notification email: %v", err)
	}

	return nil
}

func (j *SubscriptionBillingJob) StartScheduler(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("Starting subscription billing scheduler with interval: %v", interval)

	for {
		select {
		case <-ctx.Done():
			log.Println("Subscription billing scheduler stopped")
			return
		case <-ticker.C:
			if err := j.Run(ctx); err != nil {
				log.Printf("Subscription billing job failed: %v", err)
			}
		}
	}
}
