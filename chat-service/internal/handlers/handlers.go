package handlers

import (
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/graduation/chat-service/internal/models"
	"github.com/graduation/chat-service/internal/service"
	"github.com/lib/pq"
)

type Handler struct {
	svc *service.Service
}

func NewHandler(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

// --- Requests ---
type CreateDirectRequest struct {
	PeerID string `json:"peer_id"`
}

type CreateGroupRequest struct {
	Name      string   `json:"name"`
	MemberIDs []string `json:"member_ids"`
}

type SendMessageRequest struct {
	Content   string             `json:"content"`
	Type      models.MessageType `json:"type"`
	MediaURLs []string           `json:"media_urls"`
}

type TypingRequest struct {
	IsTyping bool `json:"is_typing"`
}

type AddMemberRequest struct {
	UserID string `json:"user_id"`
}

type RemoveMemberRequest struct {
	UserID string `json:"user_id"`
}

type UpdateMemberRoleRequest struct {
	UserID string     `json:"user_id"`
	Role   models.MemberRole `json:"role"`
}

// --- Handlers ---

func (h *Handler) CreateDirectConversation(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	var req CreateDirectRequest
	if err := c.BodyParser(&req); err != nil {
		fmt.Printf("BodyParser Error: %v\n", err)
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}
	fmt.Printf("CreateDirect REQ: %+v\n", req)

	conv, err := h.svc.CreateDirectConversation(userID, req.PeerID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(conv)
}

func (h *Handler) CreateGroupConversation(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	var req CreateGroupRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	conv, err := h.svc.CreateGroupConversation(userID, req.Name, req.MemberIDs)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(conv)
}

func (h *Handler) GetUserConversations(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	convs, err := h.svc.GetUserConversations(userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(convs)
}

func (h *Handler) GetConversation(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	id := c.Params("id")

	conv, err := h.svc.GetConversation(id, userID)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": "Forbidden or not found"})
	}
	return c.JSON(conv)
}

func (h *Handler) SendMessage(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	conversationID := c.Params("id")

	var req SendMessageRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	msg, err := h.svc.SendMessage(conversationID, userID, req.Content, req.Type, pq.StringArray(req.MediaURLs))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(msg)
}

func (h *Handler) GetMessages(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	conversationID := c.Params("id")

	limit := 50
	offset := 0

	if l := c.Query("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil {
			limit = val
		}
	}
	if o := c.Query("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil {
			offset = val
		}
	}

	msgs, err := h.svc.GetMessages(conversationID, userID, limit, offset)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(msgs)
}

func (h *Handler) SetTyping(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)

	var req struct {
		ConversationID string `json:"conversation_id"`
		IsTyping       bool   `json:"is_typing"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	if err := h.svc.SetTyping(req.ConversationID, userID, req.IsTyping); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(200)
}

func (h *Handler) PinMessage(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	conversationID := c.Params("id")
	messageID := c.Params("messageId")

	if err := h.svc.PinMessage(conversationID, messageID, userID); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(200)
}

func (h *Handler) UnpinMessage(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	conversationID := c.Params("id")
	messageID := c.Params("messageId")

	if err := h.svc.UnpinMessage(conversationID, messageID, userID); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(200)
}

func (h *Handler) GetPinnedMessages(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	conversationID := c.Params("id") // Note: Route structure might be messages/pinned or conversation/:id/pinned

	// Based on Router: messages.Get("/pinned", hdlrs.Message.GetPinnedMessages)
	// Which is /api/v1/conversations/:id/messages/pinned

	pins, err := h.svc.GetPinnedMessages(conversationID, userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(pins)
}

func (h *Handler) PresignMedia(c *fiber.Ctx) error {
	var req struct {
		Folder string `json:"folder"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	data := h.svc.PresignMedia(req.Folder)
	return c.JSON(data)
}

// --- Member Management ---

func (h *Handler) AddMember(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	conversationID := c.Params("id")

	var req AddMemberRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	if err := h.svc.AddMember(conversationID, userID, req.UserID); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "Member added successfully"})
}

func (h *Handler) RemoveMember(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	conversationID := c.Params("id")

	var req RemoveMemberRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	if err := h.svc.RemoveMember(conversationID, userID, req.UserID); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "Member removed successfully"})
}

func (h *Handler) UpdateMemberRole(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	conversationID := c.Params("id")

	var req UpdateMemberRoleRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	if err := h.svc.UpdateMemberRole(conversationID, userID, req.UserID, req.Role); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "Member role updated successfully"})
}

func (h *Handler) GetMembers(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	conversationID := c.Params("id")

	members, err := h.svc.GetMembers(conversationID, userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(members)
}

func (h *Handler) LeaveConversation(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	conversationID := c.Params("id")

	if err := h.svc.LeaveConversation(conversationID, userID); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "Left conversation successfully"})
}

func (h *Handler) DeleteConversation(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	conversationID := c.Params("id")

	if err := h.svc.DeleteConversation(conversationID, userID); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "Conversation deleted successfully"})
}
