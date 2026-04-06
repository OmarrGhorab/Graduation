package payment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/OmarrGhorab/payment-service/internal/domain/payment"
	"github.com/OmarrGhorab/payment-service/internal/domain/subscription"
	"github.com/OmarrGhorab/payment-service/internal/infrastructure/cache/redis"
	"github.com/OmarrGhorab/payment-service/internal/infrastructure/coursesclient"
	"github.com/OmarrGhorab/payment-service/internal/infrastructure/messaging/kafka"
	"github.com/OmarrGhorab/payment-service/internal/infrastructure/paymob"
	"github.com/OmarrGhorab/payment-service/internal/infrastructure/persistence/postgres"
	"github.com/OmarrGhorab/payment-service/internal/interfaces/http/dto"
	"github.com/google/uuid"
)


type Service struct {
	repo             *postgres.PaymentRepository
	cartRepo         *postgres.CartRepository
	subscriptionRepo *postgres.SubscriptionRepository
	paymobClient     *paymob.Client
	coursesClient    *coursesclient.Client
	redisClient      *redis.Client
	kafkaProducer    *kafka.Producer
}

func NewService(
	repo *postgres.PaymentRepository,
	cartRepo *postgres.CartRepository,
	subscriptionRepo *postgres.SubscriptionRepository,
	paymobClient *paymob.Client,
	coursesClient *coursesclient.Client,
	redisClient *redis.Client,
	kafkaProducer *kafka.Producer,
) *Service {
	return &Service{
		repo:             repo,
		cartRepo:         cartRepo,
		subscriptionRepo: subscriptionRepo,
		paymobClient:     paymobClient,
		coursesClient:    coursesClient,
		redisClient:      redisClient,
		kafkaProducer:    kafkaProducer,
	}
}

type CreatePaymentOptions struct {
	UserID        uuid.UUID
	CourseID      uuid.UUID
	PaymentMethod string // "CARD" or "WALLET"
	PhoneNumber   string // Required for WALLET
	BillingData   paymob.BillingData
}

func (s *Service) CreatePayment(ctx context.Context, opts CreatePaymentOptions) (string, uuid.UUID, error) {
	// 1. Fetch course details
	course, err := s.coursesClient.GetCourseByID(ctx, opts.CourseID.String())
	if err != nil {
		return "", uuid.Nil, fmt.Errorf("failed to fetch course: %w", err)
	}

	if !course.IsPaid || course.Price <= 0 {
		return "", uuid.Nil, errors.New("course is free or price is invalid")
	}

	// Check if user is the owner
	if course.TeacherID == opts.UserID.String() {
		return "", uuid.Nil, errors.New("you cannot buy your own course")
	}

	// Check if already enrolled
	isEnrolled, isPaid, err := s.coursesClient.CheckEnrollment(ctx, opts.UserID.String(), opts.CourseID.String())
	if err != nil {
		log.Printf("Warning: failed to check enrollment: %v", err)
	} else if isEnrolled && isPaid {
		return "", uuid.Nil, errors.New("you are already enrolled and paid for this course")
	}

	// Auto-enroll if not enrolled
	if !isEnrolled {
		if err := s.coursesClient.EnrollStudent(ctx, opts.UserID.String(), opts.CourseID.String()); err != nil {
			return "", uuid.Nil, fmt.Errorf("failed to auto-enroll student: %w", err)
		}
	}




	amountCents := int64(math.Round(course.Price * 100))

	// 2. Create local PaymentOrder
	order := &payment.PaymentOrder{
		UserID:        opts.UserID,
		AmountCents:   amountCents,
		Currency:      course.Currency,
		Status:        payment.OrderStatusPending,
		OrderType:     payment.OrderTypeSingleCourse,
		PaymentMethod: opts.PaymentMethod,
	}

	if err := s.repo.CreateOrder(ctx, order); err != nil {
		return "", uuid.Nil, fmt.Errorf("failed to create payment order: %w", err)
	}

	// Create order item for single course
	orderItem := payment.PaymentOrderItem{
		PaymentOrderID: order.ID,
		CourseID:       opts.CourseID,
		PriceCents:     amountCents,
		Currency:       course.Currency,
		BillingType:    "ONE_TIME",
	}

	if err := s.repo.CreateOrderWithItems(ctx, order, []payment.PaymentOrderItem{orderItem}); err != nil {
		return "", uuid.Nil, fmt.Errorf("failed to create order items: %w", err)
	}

	// 3. Paymob Flow
	authToken, err := s.paymobClient.Authenticate(ctx)
	if err != nil {
		return "", uuid.Nil, fmt.Errorf("paymob auth failed: %w", err)
	}

	paymobOrderID, err := s.paymobClient.CreateOrder(ctx, authToken, amountCents, course.Currency)
	if err != nil {
		return "", uuid.Nil, fmt.Errorf("paymob create order failed: %w", err)
	}

	// Update order with Paymob order ID
	paymobOrderIDStr := fmt.Sprintf("%d", paymobOrderID)
	s.repo.UpdateOrderStatus(ctx, order.ID, payment.OrderStatusPending, &paymobOrderIDStr)

	integrationID := s.paymobClient.GetCardIntegrationID()
	if opts.PaymentMethod == "WALLET" {
		integrationID = s.paymobClient.GetWalletIntegrationID()
	}

	paymentToken, err := s.paymobClient.CreatePaymentKey(ctx, authToken, paymobOrderID, amountCents, course.Currency, integrationID, opts.BillingData)
	if err != nil {
		return "", uuid.Nil, fmt.Errorf("paymob create payment key failed: %w", err)
	}

	var paymentURL string
	if opts.PaymentMethod == "WALLET" {
		paymentURL, err = s.paymobClient.PayWithWallet(ctx, paymentToken, opts.PhoneNumber)
		if err != nil {
			return "", uuid.Nil, fmt.Errorf("paymob wallet pay failed: %w", err)
		}
	} else {
		paymentURL = s.paymobClient.GetCardPaymentURL(paymentToken)
	}

	// 4. Cache session in Redis
	sessionData, _ := json.Marshal(map[string]interface{}{
		"orderID":      order.ID,
		"userID":       opts.UserID,
		"courseID":     opts.CourseID,
		"amountCents":  amountCents,
		"paymentToken": paymentToken,
	})
	s.redisClient.SetPaymentSession(ctx, order.ID.String(), string(sessionData), 30*time.Minute)

	fmt.Printf("[Payment Debug] Generated Token: %s\n", paymentToken)
	fmt.Printf("[Payment Debug] Final URL: %s\n", paymentURL)

	return paymentURL, order.ID, nil
}

func (s *Service) HandleWebhook(ctx context.Context, data map[string]interface{}, hmacHeader string) error {
	// 1. Verify HMAC
	valid, err := s.paymobClient.VerifyHMAC(hmacHeader, data)
	if err != nil {
		fmt.Printf("[Webhook Debug] Verification Error: %v\n", err)
		return err
	}
	if !valid {
		fmt.Printf("[Webhook Debug] Invalid HMAC Signature\n")
		return errors.New("invalid HMAC signature")
	}

	fmt.Printf("[Webhook Debug] HMAC Verified Successfully\n")

	// 2. Extract transaction data
	obj, ok := data["obj"].(map[string]interface{})
	if !ok {
		return errors.New("invalid webhook payload structure")
	}

	success := obj["success"].(bool)
	paymobTransactionID := fmt.Sprintf("%.0f", obj["id"])

	paymobOrderID := ""
	if o, ok := obj["order"].(map[string]interface{}); ok {
		paymobOrderID = fmt.Sprintf("%.0f", o["id"])
	} else {
		paymobOrderID = fmt.Sprintf("%.0f", obj["order"])
	}

	amountCents := int64(obj["amount_cents"].(float64))

	sourceData := obj["source_data"].(map[string]interface{})
	paymentMethod := fmt.Sprintf("%v", sourceData["sub_type"])

	// 3. Idempotency Lock
	locked, err := s.redisClient.AcquireIdempotencyLock(ctx, paymobTransactionID, 1*time.Hour)
	if err != nil || !locked {
		log.Printf("Transaction %s already being processed or lock failed", paymobTransactionID)
		return nil // Avoid retries if already processed
	}

	// 4. Fetch the order
	order, err := s.repo.GetOrderByPaymobID(ctx, paymobOrderID)
	if err != nil {
		return fmt.Errorf("failed to find order: %w", err)
	}

	if order.Status == payment.OrderStatusPaid {
		return nil // Already paid
	}

	// 5. Store Transaction
	rawResp, _ := json.Marshal(data)
	transaction := &payment.PaymentTransaction{
		PaymentOrderID:      order.ID,
		PaymobTransactionID: paymobTransactionID,
		PaymentMethod:       paymentMethod,
		AmountCents:         amountCents,
		Success:             success,
		RawResponse:         rawResp,
	}

	if err := s.repo.CreateTransaction(ctx, transaction); err != nil {
		return fmt.Errorf("failed to store transaction: %w", err)
	}

	// 6. Update Order and Activate Enrollment
	if success {
		if order.AmountCents != amountCents {
			log.Printf("CRITICAL: Amount mismatch for order %s. Expected %d, got %d", order.ID, order.AmountCents, amountCents)
			return errors.New("amount mismatch")
		}

		if err := s.repo.UpdateOrderStatus(ctx, order.ID, payment.OrderStatusPaid, nil); err != nil {
			return fmt.Errorf("failed to update order status: %w", err)
		}

		// Handle different order types
		switch order.OrderType {
		case payment.OrderTypeCartCheckout:
			// Activate all enrollments and create subscriptions for monthly items
			for _, item := range order.Items {
				if err := s.coursesClient.ActivateEnrollment(ctx, order.UserID.String(), item.CourseID.String()); err != nil {
					log.Printf("ERROR: Failed to activate enrollment for user %s, course %s: %v", order.UserID, item.CourseID, err)
				}

				// Create subscription for monthly billing
				if item.BillingType == "MONTHLY" {
					sub := &subscription.Subscription{
						UserID:          order.UserID,
						CourseID:        item.CourseID,
						Status:          subscription.StatusActive,
						PriceCents:      item.PriceCents,
						Currency:        item.Currency,
						BillingCycle:    subscription.BillingCycleMonthly,
						NextBillingDate: time.Now().AddDate(0, 1, 0),
						StartedAt:       time.Now(),
					}
					if err := s.subscriptionRepo.Create(ctx, sub); err != nil {
						log.Printf("ERROR: Failed to create subscription for user %s, course %s: %v", order.UserID, item.CourseID, err)
					}
				}
			}

			// Clear cart after successful checkout
			if err := s.cartRepo.ClearCart(ctx, order.UserID); err != nil {
				log.Printf("WARNING: Failed to clear cart for user %s: %v", order.UserID, err)
			}

		case payment.OrderTypeSubscriptionRenewal:
			// Update subscription billing date
			if order.SubscriptionID != nil {
				now := time.Now()
				nextBilling := now.AddDate(0, 1, 0)
				if err := s.subscriptionRepo.UpdateBillingDate(ctx, *order.SubscriptionID, now, nextBilling); err != nil {
					log.Printf("ERROR: Failed to update subscription billing date: %v", err)
				}
			}

		default: // OrderTypeSingleCourse (legacy)
			// For backward compatibility with old single-course orders
			if len(order.Items) > 0 {
				courseID := order.Items[0].CourseID
				if err := s.coursesClient.ActivateEnrollment(ctx, order.UserID.String(), courseID.String()); err != nil {
					log.Printf("ERROR: Failed to activate enrollment for user %s, course %s: %v", order.UserID, courseID, err)
				}
			}
		}

		// 7. Emit Kafka Event
		event := map[string]interface{}{
			"event_type": "payment.completed",
			"user_id":    order.UserID.String(),
			"payment_id": order.ID.String(),
			"amount":     amountCents,
			"currency":   order.Currency,
			"order_type": order.OrderType,
		}
		if err := s.kafkaProducer.Publish(ctx, "payments.completed.v1", order.ID.String(), event); err != nil {
			log.Printf("ERROR: Failed to publish Kafka event: %v", err)
		}

		log.Printf("SUCCESS: Payment completed for order %s, user %s", order.ID, order.UserID)
	} else {
		s.repo.UpdateOrderStatus(ctx, order.ID, payment.OrderStatusFailed, nil)
		
		// If subscription renewal failed, retry in 3 days
		if order.OrderType == payment.OrderTypeSubscriptionRenewal && order.SubscriptionID != nil {
			retryDate := time.Now().AddDate(0, 0, 3)
			// Get current last billing date
			sub, err := s.subscriptionRepo.GetByID(ctx, *order.SubscriptionID)
			if err == nil {
				lastBilling := time.Now()
				if sub.LastBillingDate != nil {
					lastBilling = *sub.LastBillingDate
				}
				s.subscriptionRepo.UpdateBillingDate(ctx, *order.SubscriptionID, lastBilling, retryDate)
			}
		}
		
		log.Printf("FAILED: Payment failed for order %s, Paymob Transaction %s", order.ID, paymobTransactionID)
	}

	return nil
}

func (s *Service) GetOrderByPaymobID(ctx context.Context, paymobOrderID string) (*payment.PaymentOrder, error) {
	return s.repo.GetOrderByPaymobID(ctx, paymobOrderID)
}

func (s *Service) GetOrderStatus(ctx context.Context, userID uuid.UUID, orderID uuid.UUID) (*payment.PaymentOrder, error) {
	order, err := s.repo.GetOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	// If a userID is provided, we must verify ownership.
	// If userID is Nil (public access), we allow the check by orderID only.
	if userID != uuid.Nil && order.UserID != userID {
		return nil, errors.New("unauthorized")
	}

	return order, nil
}

func (s *Service) GetIdempotentResponse(ctx context.Context, userID, courseID, key string) (*dto.CreatePaymentResponse, error) {
	redisKey := fmt.Sprintf("idempotency:payment:%s:%s:%s", userID, courseID, key)
	data, err := s.redisClient.Get(ctx, redisKey).Result()
	if err != nil {
		return nil, err
	}

	var resp dto.CreatePaymentResponse
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

func (s *Service) CacheIdempotentResponse(ctx context.Context, userID, courseID, key string, resp dto.CreatePaymentResponse) {
	redisKey := fmt.Sprintf("idempotency:payment:%s:%s:%s", userID, courseID, key)
	data, _ := json.Marshal(resp)
	s.redisClient.Set(ctx, redisKey, string(data), 1*time.Hour)
}


func (s *Service) GetEnrollmentStatus(ctx context.Context, userID, courseID string) (bool, bool, error) {
	return s.coursesClient.CheckEnrollment(ctx, userID, courseID)
}


