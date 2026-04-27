package http

import (
	"github.com/gofiber/fiber/v2"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) RegisterRoutes(app *fiber.App) {
	app.Get("/health", h.Health)
	app.Get("/debug-sentry", func(c *fiber.Ctx) error {
		panic("Sentry Debug Test: Payment Service is connected!")
	})
}

func (h *HealthHandler) Health(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"status":  "healthy",
			"service": "payment-service",
		},
	})
}
