package payment

import (
	"context"
	"fmt"
	"log"

	"github.com/OmarrGhorab/payment-service/internal/domain/payment"
	"github.com/OmarrGhorab/payment-service/internal/infrastructure/paymob"
	"github.com/google/uuid"
)

type RenewSubscriptionOptions struct {
	SubscriptionID uuid.UUID
	BillingData    paymob.BillingData
}

func (s *Service) RenewSubscription(ctx context.Context, opts RenewSubscriptionOptions) (string, uuid.UUID, error) {
	// 1. Get subscription
	sub, err := s.subscriptionRepo.GetByID(ctx, opts.SubscriptionID)
	if err != nil {
		return "", uuid.Nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	// 2. Create payment order for renewal
	order := &payment.PaymentOrder{
		UserID:         sub.UserID,
		AmountCents:    sub.PriceCents,
		Currency:       sub.Currency,
		Status:         payment.OrderStatusPending,
		OrderType:      payment.OrderTypeSubscriptionRenewal,
		SubscriptionID: &sub.ID,
		PaymentMethod:  "CARD", // Default to card for renewals
	}

	orderItems := []payment.PaymentOrderItem{
		{
			CourseID:    sub.CourseID,
			PriceCents:  sub.PriceCents,
			Currency:    sub.Currency,
			BillingType: "MONTHLY",
		},
	}

	if err := s.repo.CreateOrderWithItems(ctx, order, orderItems); err != nil {
		return "", uuid.Nil, fmt.Errorf("failed to create renewal order: %w", err)
	}

	// 3. Paymob Flow
	authToken, err := s.paymobClient.Authenticate(ctx)
	if err != nil {
		return "", uuid.Nil, fmt.Errorf("paymob auth failed: %w", err)
	}

	paymobOrderID, err := s.paymobClient.CreateOrder(ctx, authToken, sub.PriceCents, sub.Currency)
	if err != nil {
		return "", uuid.Nil, fmt.Errorf("paymob create order failed: %w", err)
	}

	paymobOrderIDStr := fmt.Sprintf("%d", paymobOrderID)
	s.repo.UpdateOrderStatus(ctx, order.ID, payment.OrderStatusPending, &paymobOrderIDStr)

	integrationID := s.paymobClient.GetCardIntegrationID()

	paymentToken, err := s.paymobClient.CreatePaymentKey(ctx, authToken, paymobOrderID, sub.PriceCents, sub.Currency, integrationID, opts.BillingData)
	if err != nil {
		return "", uuid.Nil, fmt.Errorf("paymob create payment key failed: %w", err)
	}

	paymentURL := s.paymobClient.GetCardPaymentURL(paymentToken)

	log.Printf("Created renewal payment for subscription %s, order %s", sub.ID, order.ID)

	return paymentURL, order.ID, nil
}
