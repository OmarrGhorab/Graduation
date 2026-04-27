package http

import (
	subscriptionApp "github.com/OmarrGhorab/payment-service/internal/application/subscription"
	"github.com/OmarrGhorab/payment-service/internal/infrastructure/authclient"
	"github.com/OmarrGhorab/payment-service/internal/interfaces/http/dto"
	"github.com/OmarrGhorab/payment-service/internal/interfaces/http/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type SubscriptionHandler struct {
	subscriptionService *subscriptionApp.Service
	authClient          *authclient.Client
}

func NewSubscriptionHandler(svc *subscriptionApp.Service, auth *authclient.Client) *SubscriptionHandler {
	return &SubscriptionHandler{
		subscriptionService: svc,
		authClient:          auth,
	}
}

func (h *SubscriptionHandler) RegisterRoutes(router fiber.Router) {
	subs := router.Group("/subscriptions", middleware.Authenticate(h.authClient))

	subs.Get("/", h.GetUserSubscriptions)
	subs.Get("/:id", h.GetSubscription)
	subs.Post("/:id/cancel", h.CancelSubscription)
}

func (h *SubscriptionHandler) GetUserSubscriptions(c *fiber.Ctx) error {
	userIDStr := c.Locals("userId").(string)
	userID, _ := uuid.Parse(userIDStr)

	subs, err := h.subscriptionService.GetUserSubscriptions(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	var response []dto.SubscriptionResponse
	for _, sub := range subs {
		var lastBilling, cancelledAt *string
		if sub.LastBillingDate != nil {
			lb := sub.LastBillingDate.Format("2006-01-02 15:04:05")
			lastBilling = &lb
		}
		if sub.CancelledAt != nil {
			ca := sub.CancelledAt.Format("2006-01-02 15:04:05")
			cancelledAt = &ca
		}

		response = append(response, dto.SubscriptionResponse{
			ID:              sub.ID.String(),
			UserID:          sub.UserID.String(),
			CourseID:        sub.CourseID.String(),
			Status:          string(sub.Status),
			PriceCents:      sub.PriceCents,
			Currency:        sub.Currency,
			BillingCycle:    string(sub.BillingCycle),
			NextBillingDate: sub.NextBillingDate.Format("2006-01-02 15:04:05"),
			LastBillingDate: lastBilling,
			StartedAt:       sub.StartedAt.Format("2006-01-02 15:04:05"),
			CancelledAt:     cancelledAt,
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    response,
	})
}

func (h *SubscriptionHandler) GetSubscription(c *fiber.Ctx) error {
	userIDStr := c.Locals("userId").(string)
	userID, _ := uuid.Parse(userIDStr)
	subscriptionID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid subscription ID",
		})
	}

	sub, err := h.subscriptionService.GetSubscription(c.Context(), userID, subscriptionID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	var lastBilling, cancelledAt *string
	if sub.LastBillingDate != nil {
		lb := sub.LastBillingDate.Format("2006-01-02 15:04:05")
		lastBilling = &lb
	}
	if sub.CancelledAt != nil {
		ca := sub.CancelledAt.Format("2006-01-02 15:04:05")
		cancelledAt = &ca
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": dto.SubscriptionResponse{
			ID:              sub.ID.String(),
			UserID:          sub.UserID.String(),
			CourseID:        sub.CourseID.String(),
			Status:          string(sub.Status),
			PriceCents:      sub.PriceCents,
			Currency:        sub.Currency,
			BillingCycle:    string(sub.BillingCycle),
			NextBillingDate: sub.NextBillingDate.Format("2006-01-02 15:04:05"),
			LastBillingDate: lastBilling,
			StartedAt:       sub.StartedAt.Format("2006-01-02 15:04:05"),
			CancelledAt:     cancelledAt,
		},
	})
}

func (h *SubscriptionHandler) CancelSubscription(c *fiber.Ctx) error {
	subscriptionIDStr := c.Params("id")

	userIDStr := c.Locals("userId").(string)
	userID, _ := uuid.Parse(userIDStr)
	subscriptionID, err := uuid.Parse(subscriptionIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid subscription ID",
		})
	}

	err = h.subscriptionService.CancelSubscription(c.Context(), userID, subscriptionID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Subscription cancelled successfully",
	})
}
