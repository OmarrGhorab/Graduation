package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/graduation/chat-service/internal/models"
	"github.com/graduation/chat-service/internal/services"
)

// MessageHandler handles message-related HTTP requests
type MessageHandler struct {
	messageSvc *services.MessageService
	pollSvc    *services.PollService
}

// NewMessageHandler creates a new MessageHandler
func NewMessageHandler(messageSvc *services.MessageService, pollSvc *services.PollService) *MessageHandler {
	return &MessageHandler{
		messageSvc: messageSvc,
		pollSvc:    pollSvc,
	}
}

// SendMessageRequest is the request body for sending a message
type SendMessageRequest struct {
	Type          models.MessageType     `json:"type"`
	Content       string                 `json:"content"`
	MediaURL      string                 `json:"media_url"`
	MediaMetadata map[string]interface{} `json:"media_metadata"`
	ReplyToID     *string                `json:"reply_to_id"`
}

// SendMessage sends a message to a conversation
func (h *MessageHandler) SendMessage(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	userRole := c.Locals("user_role").(models.UserRole)
	conversationID := c.Params("id")

	var req SendMessageRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{"code": "BAD_REQUEST", "message": "Invalid request body"},
		})
	}

	if req.Type == "" {
		req.Type = models.MessageTypeText
	}

	input := services.SendMessageInput{
		ConversationID: conversationID,
		SenderID:       userID,
		SenderRole:     userRole,
		Type:           req.Type,
		Content:        req.Content,
		MediaURL:       req.MediaURL,
		MediaMetadata:  req.MediaMetadata,
		ReplyToID:      req.ReplyToID,
	}

	message, err := h.messageSvc.SendMessage(c.Context(), input)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": fiber.Map{"code": "FORBIDDEN", "message": err.Error()},
		})
	}

	return c.Status(fiber.StatusCreated).JSON(message)
}

// GetMessages retrieves messages for a conversation
func (h *MessageHandler) GetMessages(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	conversationID := c.Params("id")
	limit := c.QueryInt("limit", 50)
	offset := c.QueryInt("offset", 0)

	messages, err := h.messageSvc.GetMessages(c.Context(), conversationID, userID, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": fiber.Map{"code": "FORBIDDEN", "message": err.Error()},
		})
	}

	return c.JSON(fiber.Map{"messages": messages})
}

// PollMessages long polls for new messages
func (h *MessageHandler) PollMessages(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	conversationID := c.Params("id")
	afterMessageID := c.Query("after", "")

	response, err := h.pollSvc.PollMessages(c.Context(), userID, conversationID, afterMessageID)
	if err != nil {
		if err.Error() == "context canceled" {
			return c.Status(fiber.StatusNoContent).Send(nil)
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{"code": "INTERNAL_ERROR", "message": err.Error()},
		})
	}

	if len(response.Messages) == 0 {
		return c.Status(fiber.StatusNoContent).Send(nil)
	}

	return c.JSON(response)
}

// DeleteMessage deletes a message
func (h *MessageHandler) DeleteMessage(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	userRole := c.Locals("user_role").(models.UserRole)
	messageID := c.Params("messageId")

	if err := h.messageSvc.DeleteMessage(c.Context(), messageID, userID, userRole); err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": fiber.Map{"code": "FORBIDDEN", "message": err.Error()},
		})
	}

	return c.JSON(fiber.Map{"message": "Message deleted successfully"})
}

// PinMessage pins a message
func (h *MessageHandler) PinMessage(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	userRole := c.Locals("user_role").(models.UserRole)
	memberRole := c.Locals("member_role").(models.MemberRole)
	conversationID := c.Params("id")
	messageID := c.Params("messageId")

	if err := h.messageSvc.PinMessage(c.Context(), conversationID, messageID, userID, userRole, memberRole); err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": fiber.Map{"code": "FORBIDDEN", "message": err.Error()},
		})
	}

	return c.JSON(fiber.Map{"message": "Message pinned successfully"})
}

// UnpinMessage unpins a message
func (h *MessageHandler) UnpinMessage(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	userRole := c.Locals("user_role").(models.UserRole)
	memberRole := c.Locals("member_role").(models.MemberRole)
	conversationID := c.Params("id")
	messageID := c.Params("messageId")

	if err := h.messageSvc.UnpinMessage(c.Context(), conversationID, messageID, userID, userRole, memberRole); err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": fiber.Map{"code": "FORBIDDEN", "message": err.Error()},
		})
	}

	return c.JSON(fiber.Map{"message": "Message unpinned successfully"})
}

// GetPinnedMessages retrieves pinned messages for a conversation
func (h *MessageHandler) GetPinnedMessages(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	conversationID := c.Params("id")

	pinnedMsgs, err := h.messageSvc.GetPinnedMessages(c.Context(), conversationID, userID)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": fiber.Map{"code": "FORBIDDEN", "message": err.Error()},
		})
	}

	return c.JSON(fiber.Map{"pinned_messages": pinnedMsgs})
}
