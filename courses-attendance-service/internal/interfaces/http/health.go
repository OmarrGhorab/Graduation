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
	app.Get("/ready", h.Ready)
}

// Health returns basic service health status
func (h *HealthHandler) Health(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"status":  "healthy",
			"service": "courses-attendance-service",
		},
	})
}

// Ready checks if service is ready to accept requests
func (h *HealthHandler) Ready(c *fiber.Ctx) error {
	// TODO: Add database and Redis connectivity checks
	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"status": "ready",
		},
	})
}
