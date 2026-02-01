package handlers

import (
	"github.com/gofiber/fiber/v2"
)

// HealthHandler handles health check HTTP requests
type HealthHandler struct{}

// NewHealthHandler creates a new HealthHandler
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// Health returns the health status of the service
func (h *HealthHandler) Health(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":  "ok",
		"service": "chat-service",
	})
}

// Ready returns the readiness status of the service
func (h *HealthHandler) Ready(c *fiber.Ctx) error {
	// TODO: Add database and Redis connectivity checks
	return c.JSON(fiber.Map{
		"status": "ready",
	})
}
