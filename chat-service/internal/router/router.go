package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/graduation/chat-service/internal/handlers"
	"github.com/graduation/chat-service/internal/middleware"
)

// SetupRoutes configures all API routes
func SetupRoutes(app *fiber.App, hdlrs *handlers.Handlers, authMiddleware *middleware.AuthMiddleware) {
	// Health routes (no auth required)
	app.Get("/health", hdlrs.Health.Health)
	app.Get("/ready", hdlrs.Health.Ready)

	// API v1 routes
	api := app.Group("/api/v1")

	// Protected routes (require authentication)
	protected := api.Group("", authMiddleware.GetMiddleware())

	// Conversation routes
	conversations := protected.Group("/conversations")
	conversations.Post("/", hdlrs.Conversation.CreateGroup)
	conversations.Post("/direct", hdlrs.Conversation.CreateDirect)
	conversations.Get("/", hdlrs.Conversation.GetUserConversations)
	conversations.Get("/:id", hdlrs.Conversation.GetConversation)
	conversations.Post("/:id/members", hdlrs.Conversation.AddMember)
	conversations.Delete("/:id/members/:memberId", hdlrs.Conversation.RemoveMember)
	conversations.Patch("/:id/members/:memberId/role", hdlrs.Conversation.UpdateMemberRole)
	conversations.Post("/:id/read", hdlrs.Conversation.MarkAsRead)

	// Message routes
	conversations.Post("/:id/messages", hdlrs.Message.SendMessage)
	conversations.Get("/:id/messages", hdlrs.Message.GetMessages)
	conversations.Get("/:id/poll", hdlrs.Message.PollMessages)
	conversations.Patch("/:id/messages/:messageId", hdlrs.Message.EditMessage)
	conversations.Delete("/:id/messages/:messageId", hdlrs.Message.DeleteMessage)
	conversations.Post("/:id/messages/:messageId/pin", hdlrs.Message.PinMessage)
	conversations.Delete("/:id/messages/:messageId/pin", hdlrs.Message.UnpinMessage)
	conversations.Get("/:id/pinned", hdlrs.Message.GetPinnedMessages)

	// Typing indicator routes
	protected.Post("/typing", hdlrs.Typing.SetTyping)
	protected.Get("/typing", hdlrs.Typing.GetTypingUsers)

	// Media routes
	protected.Post("/media/presign", hdlrs.Media.Presign)
	protected.Post("/media/batch-presign", hdlrs.Media.BatchPresign)
}
