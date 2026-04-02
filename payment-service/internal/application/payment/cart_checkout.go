package payment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	cartDomain "github.com/OmarrGhorab/payment-service/internal/domain/cart"
	"github.com/OmarrGhorab/payment-service/internal/domain/payment"
	"github.com/OmarrGhorab/payment-service/internal/infrastructure/paymob"
	"github.com/google/uuid"
)

type CheckoutCartOptions struct {
	UserID        uuid.UUID
	PaymentMethod string
	PhoneNumber   string
	BillingData   paymob.BillingData
}

func (s *Service) CheckoutCart(ctx context.Context, opts CheckoutCartOptions) (string, uuid.UUID, error) {
	// 1. Get cart with items
	userCart, err := s.cartRepo.GetCartWithItems(ctx, opts.UserID)
	if err != nil {
		return "", uuid.Nil, fmt.Errorf("failed to get cart: %w", err)
	}

	if len(userCart.Items) == 0 {
		return "", uuid.Nil, errors.New("cart is empty")
	}

	// 2. Validate all courses and check enrollments
	var totalAmount int64
	currency := userCart.Items[0].Currency
	var orderItems []payment.PaymentOrderItem
	var monthlyItems []cartDomain.CartItem

	for _, item := range userCart.Items {
		if item.Currency != currency {
			return "", uuid.Nil, errors.New("mixed currencies in cart not supported")
		}

		// Validate course still exists and is paid
		course, err := s.coursesClient.GetCourseByID(ctx, item.CourseID.String())
		if err != nil {
			return "", uuid.Nil, fmt.Errorf("failed to fetch course %s: %w", item.CourseID, err)
		}

		if !course.IsPaid {
			return "", uuid.Nil, fmt.Errorf("course %s is no longer a paid course", item.CourseID)
		}

		// Check if already enrolled and paid
		isEnrolled, isPaid, err := s.coursesClient.CheckEnrollment(ctx, opts.UserID.String(), item.CourseID.String())
		if err == nil && isEnrolled && isPaid {
			return "", uuid.Nil, fmt.Errorf("you are already enrolled and paid for course %s", item.CourseID)
		}

		// Auto-enroll if not enrolled
		if !isEnrolled {
			if err := s.coursesClient.EnrollStudent(ctx, opts.UserID.String(), item.CourseID.String()); err != nil {
				return "", uuid.Nil, fmt.Errorf("failed to auto-enroll in course %s: %w", item.CourseID, err)
			}
		}

		totalAmount += item.PriceCents

		orderItems = append(orderItems, payment.PaymentOrderItem{
			CourseID:    item.CourseID,
			PriceCents:  item.PriceCents,
			Currency:    item.Currency,
			BillingType: string(item.BillingType),
		})

		if item.BillingType == cartDomain.BillingTypeMonthly {
			monthlyItems = append(monthlyItems, item)
		}
	}

	// 3. Create payment order
	order := &payment.PaymentOrder{
		UserID:        opts.UserID,
		AmountCents:   totalAmount,
		Currency:      currency,
		Status:        payment.OrderStatusPending,
		OrderType:     payment.OrderTypeCartCheckout,
		PaymentMethod: opts.PaymentMethod,
	}

	if err := s.repo.CreateOrderWithItems(ctx, order, orderItems); err != nil {
		return "", uuid.Nil, fmt.Errorf("failed to create payment order: %w", err)
	}

	// 4. Paymob Flow
	authToken, err := s.paymobClient.Authenticate(ctx)
	if err != nil {
		return "", uuid.Nil, fmt.Errorf("paymob auth failed: %w", err)
	}

	paymobOrderID, err := s.paymobClient.CreateOrder(ctx, authToken, totalAmount, currency)
	if err != nil {
		return "", uuid.Nil, fmt.Errorf("paymob create order failed: %w", err)
	}

	paymobOrderIDStr := fmt.Sprintf("%d", paymobOrderID)
	s.repo.UpdateOrderStatus(ctx, order.ID, payment.OrderStatusPending, &paymobOrderIDStr)

	integrationID := s.paymobClient.GetCardIntegrationID()
	if opts.PaymentMethod == "WALLET" {
		integrationID = s.paymobClient.GetWalletIntegrationID()
	}

	paymentToken, err := s.paymobClient.CreatePaymentKey(ctx, authToken, paymobOrderID, totalAmount, currency, integrationID, opts.BillingData)
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

	// 5. Cache session in Redis
	sessionData, _ := json.Marshal(map[string]interface{}{
		"orderID":      order.ID,
		"userID":       opts.UserID,
		"amountCents":  totalAmount,
		"paymentToken": paymentToken,
		"monthlyItems": monthlyItems,
	})
	s.redisClient.SetPaymentSession(ctx, order.ID.String(), string(sessionData), 30*time.Minute)

	return paymentURL, order.ID, nil
}
