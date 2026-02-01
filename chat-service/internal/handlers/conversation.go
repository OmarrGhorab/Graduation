package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/graduation/chat-service/internal/models"
	"github.com/graduation/chat-service/internal/repositories"
	"github.com/graduation/chat-service/internal/services"
)

// ConversationHandler handles conversation-related HTTP requests
type ConversationHandler struct {
	conversationSvc *services.ConversationService
}

// NewConversationHandler creates a new ConversationHandler
func NewConversationHandler(conversationSvc *services.ConversationService) *ConversationHandler {
	return &ConversationHandler{conversationSvc: conversationSvc}
}

// CreateGroupRequest is the request body for creating a group chat
type CreateGroupRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	MemberIDs   []string `json:"member_ids"`
}

// CreateGroup creates a new group conversation
func (h *ConversationHandler) CreateGroup(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	userRole := c.Locals("user_role").(models.UserRole)

	var req CreateGroupRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{"code": "BAD_REQUEST", "message": "Invalid request body"},
		})
	}

	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{"code": "BAD_REQUEST", "message": "Group name is required"},
		})
	}

	input := services.CreateGroupInput{
		Name:        req.Name,
		Description: req.Description,
		MemberIDs:   req.MemberIDs,
		CreatorID:   userID,
		CreatorRole: userRole,
	}

	conversation, err := h.conversationSvc.CreateGroup(c.Context(), input)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": fiber.Map{"code": "FORBIDDEN", "message": err.Error()},
		})
	}

	return c.Status(fiber.StatusCreated).JSON(conversation)
}

// CreateDirectRequest is the request body for creating a direct chat
type CreateDirectRequest struct {
	RecipientID string `json:"recipient_id"`
}

// CreateDirect creates a direct conversation
func (h *ConversationHandler) CreateDirect(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	userRole := c.Locals("user_role").(models.UserRole)

	var req CreateDirectRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{"code": "BAD_REQUEST", "message": "Invalid request body"},
		})
	}

	if req.RecipientID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{"code": "BAD_REQUEST", "message": "Recipient ID is required"},
		})
	}

	conversation, err := h.conversationSvc.CreateDirectChat(c.Context(), userID, req.RecipientID, userRole)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{"code": "INTERNAL_ERROR", "message": err.Error()},
		})
	}

	return c.Status(fiber.StatusCreated).JSON(conversation)
}

// GetConversation retrieves a conversation by ID
func (h *ConversationHandler) GetConversation(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	conversationID := c.Params("id")

	// Check membership
	isMember, err := h.conversationSvc.IsMember(c.Context(), conversationID, userID)
	if err != nil || !isMember {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": fiber.Map{"code": "FORBIDDEN", "message": "You are not a member of this conversation"},
		})
	}

	conversation, err := h.conversationSvc.GetByID(c.Context(), conversationID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": fiber.Map{"code": "NOT_FOUND", "message": "Conversation not found"},
		})
	}

	return c.JSON(conversation)
}

// MarkAsRead marks all messages in a conversation as read
func (h *ConversationHandler) MarkAsRead(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	conversationID := c.Params("id")

	if err := h.conversationSvc.MarkAsRead(c.Context(), conversationID, userID); err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": fiber.Map{"code": "FORBIDDEN", "message": err.Error()},
		})
	}

	return c.JSON(fiber.Map{"message": "Conversation marked as read"})
}

// GetUserConversations retrieves all conversations for the current user
func (h *ConversationHandler) GetUserConversations(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	limit := c.QueryInt("limit", 20)
	offset := c.QueryInt("offset", 0)

	filter := repositories.ConversationFilter{
		Role:  models.UserRole(c.Query("role")),
		Type:  models.ConversationType(c.Query("type")),
		Query: c.Query("q"),
	}
	if filter.Query == "" {
		filter.Query = c.Query("query")
	}

	conversations, err := h.conversationSvc.GetUserConversations(c.Context(), userID, filter, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{"code": "INTERNAL_ERROR", "message": err.Error()},
		})
	}

	return c.JSON(fiber.Map{"conversations": conversations})
}

// AddMemberRequest is the request body for adding a member
type AddMemberRequest struct {
	UserID     string            `json:"user_id"`
	MemberRole models.MemberRole `json:"member_role"`
}

// AddMember adds a member to a conversation
func (h *ConversationHandler) AddMember(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	conversationID := c.Params("id")

	var req AddMemberRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{"code": "BAD_REQUEST", "message": "Invalid request body"},
		})
	}

	if req.MemberRole == "" {
		req.MemberRole = models.MemberRoleMember
	}

	if err := h.conversationSvc.AddMember(c.Context(), conversationID, userID, req.UserID, req.MemberRole); err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": fiber.Map{"code": "FORBIDDEN", "message": err.Error()},
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "Member added successfully"})
}

// RemoveMember removes a member from a conversation
func (h *ConversationHandler) RemoveMember(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	conversationID := c.Params("id")
	memberID := c.Params("memberId")

	if err := h.conversationSvc.RemoveMember(c.Context(), conversationID, userID, memberID); err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": fiber.Map{"code": "FORBIDDEN", "message": err.Error()},
		})
	}

	return c.JSON(fiber.Map{"message": "Member removed successfully"})
}

// UpdateMemberRoleRequest is the request body for updating member role
type UpdateMemberRoleRequest struct {
	MemberRole models.MemberRole `json:"member_role"`
}

// UpdateMemberRole updates a member's role
func (h *ConversationHandler) UpdateMemberRole(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	conversationID := c.Params("id")
	memberID := c.Params("memberId")

	var req UpdateMemberRoleRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{"code": "BAD_REQUEST", "message": "Invalid request body"},
		})
	}

	if err := h.conversationSvc.UpdateMemberRole(c.Context(), conversationID, userID, memberID, req.MemberRole); err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": fiber.Map{"code": "FORBIDDEN", "message": err.Error()},
		})
	}

	return c.JSON(fiber.Map{"message": "Member role updated successfully"})
}
