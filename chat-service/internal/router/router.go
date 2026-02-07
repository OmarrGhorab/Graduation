package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/graduation/chat-service/internal/handlers"
	"github.com/graduation/chat-service/internal/middleware"
)

func SetupRoutes(app *fiber.App, h *handlers.Handler, auth *middleware.AuthMiddleware) {
	api := app.Group("/api/v1")

	// Conversations
	conversations := api.Group("/conversations", auth.Protected())
	conversations.Post("/", h.CreateGroupConversation)
	conversations.Post("/direct", h.CreateDirectConversation)
	conversations.Get("/", h.GetUserConversations)
	conversations.Get("/:id", h.GetConversation)
	conversations.Delete("/:id", h.DeleteConversation)
	conversations.Post("/:id/leave", h.LeaveConversation)

	// Members
	conversations.Get("/:id/members", h.GetMembers)
	conversations.Post("/:id/members", h.AddMember)
	conversations.Delete("/:id/members", h.RemoveMember)
	conversations.Put("/:id/members/role", h.UpdateMemberRole)

	// Messages
	messages := conversations.Group("/:id/messages")
	messages.Post("/", h.SendMessage)
	messages.Get("/", h.GetMessages)
	messages.Delete("/:messageId", h.DeleteMessage)

	// Read Receipts
	conversations.Post("/:id/read", h.MarkAsRead)
	conversations.Get("/:id/unread", h.GetUnreadCount)
	conversations.Post("/:id/messages/:messageId/read", h.MarkMessageAsRead)

	// Message Actions (Pinning)
	messages.Get("/pinned", h.GetPinnedMessages)
	conversations.Post("/:id/messages/:messageId/pin", h.PinMessage)
	conversations.Delete("/:id/messages/:messageId/pin", h.UnpinMessage)

	// Media Collection
	conversations.Get("/:id/media", h.GetMediaCollection)

	// Typing
	api.Post("/typing", auth.Protected(), h.SetTyping)

	// Media
	api.Post("/media/presign", auth.Protected(), h.PresignMedia)
}
