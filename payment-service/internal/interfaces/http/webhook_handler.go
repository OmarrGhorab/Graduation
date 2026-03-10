package http

import (
	"github.com/OmarrGhorab/payment-service/internal/application/payment"
	"github.com/gofiber/fiber/v2"
)

type WebhookHandler struct {
	service *payment.Service
}

func NewWebhookHandler(svc *payment.Service) *WebhookHandler {
	return &WebhookHandler{service: svc}
}

func (h *WebhookHandler) RegisterRoutes(router fiber.Router) {
	router.Post("/webhook/paymob", h.PaymobWebhook)
}

func (h *WebhookHandler) PaymobWebhook(c *fiber.Ctx) error {
	hmacHeader := c.Get("hmac")
	if hmacHeader == "" {
		return c.Status(fiber.StatusUnauthorized).SendString("HMAC header missing")
	}

	var data map[string]interface{}
	if err := c.BodyParser(&data); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid body")
	}

	if err := h.service.HandleWebhook(c.Context(), data, hmacHeader); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}

	return c.Status(fiber.StatusOK).SendString("OK")
}
