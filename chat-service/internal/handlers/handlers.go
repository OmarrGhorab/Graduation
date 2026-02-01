package handlers

import (
	"github.com/graduation/chat-service/internal/services"
)

// Handlers holds all handler instances
type Handlers struct {
	Conversation *ConversationHandler
	Message      *MessageHandler
	Typing       *TypingHandler
	Media        *MediaHandler
	Health       *HealthHandler
}

// NewHandlers creates a new Handlers instance with all handlers
func NewHandlers(svcs *services.Services) *Handlers {
	return &Handlers{
		Conversation: NewConversationHandler(svcs.Conversation),
		Message:      NewMessageHandler(svcs.Message, svcs.Poll),
		Typing:       NewTypingHandler(svcs.Typing),
		Media:        NewMediaHandler(svcs.Media),
		Health:       NewHealthHandler(),
	}
}
