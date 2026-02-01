package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/graduation/chat-service/internal/models"
	"github.com/graduation/chat-service/internal/services"
)

// TypingHandler handles typing indicator HTTP requests
type TypingHandler struct {
	typingSvc *services.TypingService
}

// NewTypingHandler creates a new TypingHandler
func NewTypingHandler(typingSvc *services.TypingService) *TypingHandler {
	return &TypingHandler{typingSvc: typingSvc}
}

// SetTypingRequest is the request body for setting typing indicator
type SetTypingRequest struct {
	ConversationID string `json:"conversation_id"`
}

// SetTyping sets the typing indicator
func (h *TypingHandler) SetTyping(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	userRole := c.Locals("user_role").(models.UserRole)

	var req SetTypingRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{"code": "BAD_REQUEST", "message": "Invalid request body"},
		})
	}

	if req.ConversationID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{"code": "BAD_REQUEST", "message": "Conversation ID is required"},
		})
	}

	if err := h.typingSvc.SetTyping(c.Context(), req.ConversationID, userID, userRole); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{"code": "INTERNAL_ERROR", "message": err.Error()},
		})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// GetTypingUsers gets users currently typing in a conversation
func (h *TypingHandler) GetTypingUsers(c *fiber.Ctx) error {
	conversationID := c.Query("conversation_id")

	if conversationID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{"code": "BAD_REQUEST", "message": "Conversation ID is required"},
		})
	}

	typingUsers, err := h.typingSvc.GetTypingUsers(c.Context(), conversationID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{"code": "INTERNAL_ERROR", "message": err.Error()},
		})
	}

	return c.JSON(fiber.Map{"typing_users": typingUsers})
}
