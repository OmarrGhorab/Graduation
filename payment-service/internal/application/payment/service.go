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
	"github.com/OmarrGhorab/payment-service/internal/domain/paymentmethod"
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
	repo                *postgres.PaymentRepository
	cartRepo            *postgres.CartRepository
	subscriptionRepo    *postgres.SubscriptionRepository
	paymentMethodRepo   *postgres.PaymentMethodRepository
	paymobClient        *paymob.Client
	coursesClient       *coursesclient.Client
	redisClient         *redis.Client
	kafkaProducer       *kafka.Producer
}

func NewService(
	repo *postgres.PaymentRepository,
	cartRepo *postgres.CartRepository,
	subscriptionRepo *postgres.SubscriptionRepository,
	paymentMethodRepo *postgres.PaymentMethodRepository,
	paymobClient *paymob.Client,
	coursesClient *coursesclient.Client,
	redisClient *redis.Client,
	kafkaProducer *kafka.Producer,
) *Service {
	return &Service{
		repo:                repo,
		cartRepo:            cartRepo,
		subscriptionRepo:    subscriptionRepo,
		paymentMethodRepo:   paymentMethodRepo,
		paymobClient:        paymobClient,
		coursesClient:       coursesClient,
		redisClient:         redisClient,
		kafkaProducer:       kafkaProducer,
	}
}

type PurchaseHistoryItem struct {
	OrderID        uuid.UUID                   `json:"orderId"`
	AmountCents    int64                       `json:"amountCents"`
	Currency       string                      `json:"currency"`
	Status         payment.OrderStatus         `json:"status"`
	OrderType      payment.OrderType           `json:"orderType"`
	CreatedAt      time.Time                   `json:"createdAt"`
	PaymentMethod  *paymentmethod.PaymentMethod `json:"paymentMethod,omitempty"`
	Items          []PurchaseHistoryCourseItem `json:"items"`
}

type PurchaseHistoryCourseItem struct {
	CourseID   uuid.UUID `json:"courseId"`
	Title      string    `json:"title"`
	PriceCents int64     `json:"priceCents"`
}

func (s *Service) GetUserPurchaseHistory(ctx context.Context, userID uuid.UUID) ([]PurchaseHistoryItem, error) {
	orders, err := s.repo.GetUserOrders(ctx, userID)
	if err != nil {
		return nil, err
	}

	history := make([]PurchaseHistoryItem, 0, len(orders))
	for _, order := range orders {
		item := PurchaseHistoryItem{
			OrderID:     order.ID,
			AmountCents:  order.AmountCents,
			Currency:    order.Currency,
			Status:      order.Status,
			OrderType:   order.OrderType,
			CreatedAt:   order.CreatedAt,
			Items:       make([]PurchaseHistoryCourseItem, 0, len(order.Items)),
		}

		// Fetch payment method if available
		if order.PaymentMethodID != nil {
			pm, err := s.paymentMethodRepo.GetByID(ctx, *order.PaymentMethodID)
			if err == nil {
				item.PaymentMethod = pm
			}
		}

		// Fetch course details for each item
		for _, orderItem := range order.Items {
			course, err := s.coursesClient.GetCourseByID(ctx, orderItem.CourseID.String())
			courseTitle := "Unknown Course"
			if err == nil {
				courseTitle = course.Title
			}

			item.Items = append(item.Items, PurchaseHistoryCourseItem{
				CourseID:   orderItem.CourseID,
				Title:      courseTitle,
				PriceCents: orderItem.PriceCents,
			})
		}

		history = append(history, item)
	}

	return history, nil
}

type CreatePaymentOptions struct {
	UserID          uuid.UUID
	CourseID        uuid.UUID
	PaymentMethod   string
	PaymentMethodID uuid.UUID // New: for one-click pay
	PhoneNumber     string
	BillingData     paymob.BillingData
	SaveCard        bool
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

	// 2. Create order item for single course
	orderItem := payment.PaymentOrderItem{
		CourseID:    opts.CourseID,
		PriceCents:  amountCents,
		Currency:    course.Currency,
		BillingType: course.BillingType,
	}

	// 3. Create local PaymentOrder with items (This handles both order and items in one transaction)
	order := &payment.PaymentOrder{
		ID:            uuid.New(),
		UserID:        opts.UserID,
		AmountCents:   amountCents,
		Currency:      course.Currency,
		Status:        payment.OrderStatusPending,
		OrderType:     payment.OrderTypeSingleCourse,
		PaymentMethod: opts.PaymentMethod,
	}

	if err := s.repo.CreateOrderWithItems(ctx, order, []payment.PaymentOrderItem{orderItem}); err != nil {
		return "", uuid.Nil, fmt.Errorf("failed to create order items: %w", err)
	}
	order.Items = []payment.PaymentOrderItem{orderItem}

	// 4. Paymob Flow
	authToken, err := s.paymobClient.Authenticate(ctx)
	if err != nil {
		return "", uuid.Nil, fmt.Errorf("paymob auth failed: %w", err)
	}

	paymobOrderID, err := s.paymobClient.CreateOrder(ctx, authToken, amountCents, course.Currency)
	if err != nil {
		return "", uuid.Nil, fmt.Errorf("paymob order creation failed: %w", err)
	}

	// Update order with paymob ID
	paymobOrderIDStr := fmt.Sprintf("%d", paymobOrderID)
	if err := s.repo.UpdateOrderStatus(ctx, order.ID, payment.OrderStatusPending, &paymobOrderIDStr); err != nil {
		return "", uuid.Nil, fmt.Errorf("failed to update order: %w", err)
	}

	// Create Payment Key
	integrationID := s.paymobClient.GetCardIntegrationID()
	if opts.PaymentMethod == "WALLET" {
		integrationID = s.paymobClient.GetWalletIntegrationID()
	}

	paymentToken, err := s.paymobClient.CreatePaymentKey(ctx, authToken, paymobOrderID, amountCents, course.Currency, integrationID, opts.BillingData, opts.SaveCard)
	if err != nil {
		return "", uuid.Nil, fmt.Errorf("paymob payment key creation failed: %w", err)
	}

	var paymentURL string

	// 5. Handle One-Click or Regular flow
	if opts.PaymentMethod == "CARD" && opts.PaymentMethodID != uuid.Nil {
		pm, err := s.paymentMethodRepo.GetByID(ctx, opts.PaymentMethodID)
		if err != nil {
			return "", uuid.Nil, fmt.Errorf("payment method not found: %w", err)
		}
		if pm.UserID != opts.UserID {
			return "", uuid.Nil, errors.New("unauthorized payment method")
		}

		resp, err := s.paymobClient.PayWithToken(ctx, paymentToken, pm.Token)
		if err != nil {
			return "", uuid.Nil, fmt.Errorf("one-click payment failed: %w", err)
		}

		paymentURL = resp.RedirectURL
		if paymentURL == "" {
			paymentURL = resp.IframeRedirectURL
		}

		// Update order status if immediately known
		if bool(resp.Success) {
			if err := s.repo.UpdateOrderStatus(ctx, order.ID, payment.OrderStatusPaid, nil); err == nil {
				s.completeSuccessfulPayment(ctx, order, amountCents)
			}
		} else if !bool(resp.Pending) {
			s.repo.UpdateOrderStatus(ctx, order.ID, payment.OrderStatusFailed, nil)
		}
	} else if opts.PaymentMethod == "WALLET" {
		paymentURL, err = s.paymobClient.PayWithWallet(ctx, paymentToken, opts.PhoneNumber)
		if err != nil {
			return "", uuid.Nil, fmt.Errorf("paymob wallet pay failed: %w", err)
		}
	} else {
		paymentURL = s.paymobClient.GetCardPaymentURL(paymentToken)
	}

	// 6. Cache session in Redis
	sessionData, _ := json.Marshal(map[string]interface{}{
		"orderID":      order.ID,
		"userID":       opts.UserID,
		"courseID":     opts.CourseID,
		"amountCents":  amountCents,
		"paymentToken": paymentToken,
		"saveCard":     opts.SaveCard,
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
	eventType, _ := data["type"].(string)
	fmt.Printf("[Webhook Debug] Event Type: %s\n", eventType)

	obj, ok := data["obj"].(map[string]interface{})
	if !ok {
		return errors.New("invalid webhook payload structure: 'obj' not found")
	}

	// Handle TOKEN specific event
	if eventType == "TOKEN" {
		token, _ := obj["token"].(string)
		pan, _ := obj["masked_pan"].(string)
		cardSubType, _ := obj["card_subtype"].(string)
		paymobOrderID := fmt.Sprintf("%v", obj["order_id"])

		log.Printf("[Webhook Debug] Processing TOKEN: %s, Order: %s", pan, paymobOrderID)

		// Find the order to get the UserID
		order, err := s.repo.GetOrderByPaymobID(ctx, paymobOrderID)
		if err != nil {
			return fmt.Errorf("failed to find order for token: %w", err)
		}

		lastFour := ""
		if len(pan) >= 4 {
			lastFour = pan[len(pan)-4:]
		}

		pm := &paymentmethod.PaymentMethod{
			ID:          uuid.New(),
			UserID:      order.UserID,
			PaymentType: paymentmethod.PaymentTypeCard,
			Token:       token,
			LastFour:    lastFour,
			CardBrand:   cardSubType,
			IsActive:    true,
			IsDefault:   true,
		}

		if err := s.paymentMethodRepo.Create(ctx, pm); err != nil {
			log.Printf("WARNING: Failed to save payment method from TOKEN event: %v", err)
			return nil // Don't fail the webhook if PM save fails but it's a valid event
		}

		log.Printf("SUCCESS: Saved payment method %s (****%s) for user %s via TOKEN event", cardSubType, lastFour, order.UserID)
		return nil // Done with TOKEN event
	}

	// For TRANSACTION events (existing logic)
	success, _ := obj["success"].(bool)
	paymobTransactionID := fmt.Sprintf("%.0f", obj["id"])

	paymobOrderID := ""
	if o, ok := obj["order"].(map[string]interface{}); ok {
		paymobOrderID = fmt.Sprintf("%.0f", o["id"])
	} else {
		paymobOrderID = fmt.Sprintf("%.0f", obj["order"])
	}

	amountCents := int64(obj["amount_cents"].(float64))

	sourceData, _ := obj["source_data"].(map[string]interface{})
	paymentMethodType := "CARD"
	if sourceData != nil {
		paymentMethodType = fmt.Sprintf("%v", sourceData["sub_type"])
	}

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
		PaymentMethod:       paymentMethodType,
		AmountCents:         amountCents,
		Success:             success,
		RawResponse:         rawResp,
	}

	if err := s.repo.CreateTransaction(ctx, transaction); err != nil {
		return fmt.Errorf("failed to store transaction: %w", err)
	}

	// 6. Save Payment Method if tokenization was requested and successful (Fallback/Alternative during TRANSACTION)
	if success && sourceData != nil {
		token, _ := obj["token"].(string)
		if token == "" {
			token, _ = obj["payment_token"].(string)
		}

		if token != "" {
			pan := fmt.Sprintf("%v", sourceData["pan"])
			lastFour := ""
			if len(pan) >= 4 {
				lastFour = pan[len(pan)-4:]
			}

			brand := fmt.Sprintf("%v", sourceData["sub_type"])
			
			pm := &paymentmethod.PaymentMethod{
				ID:          uuid.New(),
				UserID:      order.UserID,
				PaymentType: paymentmethod.PaymentTypeCard,
				Token:       token,
				LastFour:    lastFour,
				CardBrand:   brand,
				IsActive:    true,
				IsDefault:   true,
			}
			
			// Check if already exists to avoid duplicates if TOKEN event already fired
			existing, _ := s.paymentMethodRepo.GetUserPaymentMethods(ctx, order.UserID)
			alreadyExists := false
			for _, e := range existing {
				if e.Token == token {
					alreadyExists = true
					break
				}
			}

			if !alreadyExists {
				if err := s.paymentMethodRepo.Create(ctx, pm); err != nil {
					log.Printf("WARNING: Failed to save payment method for user %s: %v", order.UserID, err)
				} else {
					log.Printf("SUCCESS: Saved payment method %s (****%s) for user %s during TRANSACTION", brand, lastFour, order.UserID)
				}
			}
		}
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

		// 6. Complete the payment process (enrollment, subscriptions, etc.)
		if err := s.completeSuccessfulPayment(ctx, order, amountCents); err != nil {
			log.Printf("ERROR: Failed to complete payment actions for order %s: %v", order.ID, err)
		}
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

func (s *Service) GetUserPaymentMethods(ctx context.Context, userID uuid.UUID) ([]paymentmethod.PaymentMethod, error) {
	return s.paymentMethodRepo.GetUserPaymentMethods(ctx, userID)
}

func (s *Service) completeSuccessfulPayment(ctx context.Context, order *payment.PaymentOrder, amountCents int64) error {
	// 1. Activate Enrollments / Create Subscriptions
	switch order.OrderType {
	case payment.OrderTypeCartCheckout:
		for _, item := range order.Items {
			if err := s.coursesClient.ActivateEnrollment(ctx, order.UserID.String(), item.CourseID.String()); err != nil {
				log.Printf("ERROR: Failed to activate enrollment for user %s, course %s: %v", order.UserID, item.CourseID, err)
			}

			if item.BillingType == "MONTHLY" {
				s.ensureSubscription(ctx, order.UserID, item)
			}
		}
		if err := s.cartRepo.ClearCart(ctx, order.UserID); err != nil {
			log.Printf("WARNING: Failed to clear cart for user %s: %v", order.UserID, err)
		}

	case payment.OrderTypeSubscriptionRenewal:
		if order.SubscriptionID != nil {
			now := time.Now()
			nextBilling := now.AddDate(0, 1, 0)
			if err := s.subscriptionRepo.UpdateBillingDate(ctx, *order.SubscriptionID, now, nextBilling); err != nil {
				log.Printf("ERROR: Failed to update subscription: %v", err)
			}
		}

	default: // OrderTypeSingleCourse
		for _, item := range order.Items {
			if err := s.coursesClient.ActivateEnrollment(ctx, order.UserID.String(), item.CourseID.String()); err != nil {
				log.Printf("ERROR: Failed to activate enrollment for user %s, course %s: %v", order.UserID, item.CourseID, err)
			}

			if item.BillingType == "MONTHLY" {
				s.ensureSubscription(ctx, order.UserID, item)
			}
		}
	}

	// 2. Emit Kafka Event
	event := map[string]interface{}{
		"event_type": "payment.completed",
		"user_id":    order.UserID.String(),
		"payment_id": order.ID.String(),
		"amount":     amountCents,
		"currency":   order.Currency,
		"order_type": string(order.OrderType),
	}
	if err := s.kafkaProducer.Publish(ctx, "payments.completed.v1", order.ID.String(), event); err != nil {
		log.Printf("ERROR: Failed to publish Kafka event: %v", err)
	}

	log.Printf("SUCCESS: Payment completed for order %s, user %s", order.ID, order.UserID)
	return nil
}

func (s *Service) ensureSubscription(ctx context.Context, userID uuid.UUID, item payment.PaymentOrderItem) {
	// Check if already exists to avoid unique constraint error
	existing, _ := s.subscriptionRepo.GetByUserAndCourse(ctx, userID, item.CourseID)
	if existing != nil {
		if existing.Status != subscription.StatusActive {
			s.subscriptionRepo.UpdateStatus(ctx, existing.ID, subscription.StatusActive)
		}
		return
	}

	sub := &subscription.Subscription{
		ID:              uuid.New(),
		UserID:          userID,
		CourseID:        item.CourseID,
		Status:          subscription.StatusActive,
		PriceCents:      item.PriceCents,
		Currency:        item.Currency,
		BillingCycle:    subscription.BillingCycleMonthly,
		NextBillingDate: time.Now().AddDate(0, 1, 0),
		StartedAt:       time.Now(),
	}
	if err := s.subscriptionRepo.Create(ctx, sub); err != nil {
		log.Printf("ERROR: Failed to create subscription for user %s, course %s: %v", userID, item.CourseID, err)
	}
}



