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
	conversations.Get("/:id/members", hdlrs.Conversation.GetMembers)
	conversations.Patch("/:id/image", hdlrs.Conversation.UpdateImage)
	conversations.Post("/:id/members", hdlrs.Conversation.AddMember)
	conversations.Delete("/:id/members/:memberId", hdlrs.Conversation.RemoveMember)
	conversations.Patch("/:id/members/:memberId/role", hdlrs.Conversation.UpdateMemberRole)
	conversations.Post("/:id/read", hdlrs.Conversation.MarkAsRead)
	conversations.Delete("/:id", hdlrs.Conversation.DeleteConversation)

	// Message routes
	messages := conversations.Group("/:id/messages")
	messages.Post("/", hdlrs.Message.SendMessage)
	messages.Get("/", hdlrs.Message.GetMessages)
	messages.Get("/media", hdlrs.Message.GetMediaHistory)
	messages.Get("/poll", hdlrs.Message.PollMessages)
	messages.Get("/pinned", hdlrs.Message.GetPinnedMessages)
	conversations.Patch("/:id/messages/:messageId", hdlrs.Message.EditMessage)
	conversations.Delete("/:id/messages/:messageId", hdlrs.Message.DeleteMessage)
	conversations.Post("/:id/messages/:messageId/pin", hdlrs.Message.PinMessage)
	conversations.Delete("/:id/messages/:messageId/pin", hdlrs.Message.UnpinMessage)

	// Typing indicator routes
	protected.Post("/typing", hdlrs.Typing.SetTyping)
	protected.Get("/typing", hdlrs.Typing.GetTypingUsers)

	// Media routes
	protected.Post("/media/presign", hdlrs.Media.Presign)
	protected.Post("/media/batch-presign", hdlrs.Media.BatchPresign)
}
