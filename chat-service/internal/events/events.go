package events

import "time"

// Kafka Topics
const (
	TopicMessageCreated = "chat.message.created"
	TopicTyping         = "chat.typing"
	TopicReadReceipt    = "chat.read.receipt"
)

// MessageCreatedEvent is published when a message is successfully saved.
// Key: ConversationID
type MessageCreatedEvent struct {
	ID             string    `json:"id"`
	LocalID        string    `json:"local_id,omitempty"`
	ConversationID string    `json:"conversation_id"`
	SenderID       string    `json:"sender_id"`
	Content        string    `json:"content"`
	Type           string    `json:"type"` // text, image, voice
	MediaURLs      []string  `json:"media_urls,omitempty"`
	CreatedAt      time.Time `json:"created_at"`

	// Routing Info (CRITICAL for Gateway)
	RecipientIDs []string `json:"recipient_ids"`
}

// TypingEvent is published when a user starts/stops typing
// Key: ConversationID
type TypingEvent struct {
	ConversationID string   `json:"conversation_id"`
	UserID         string   `json:"user_id"`
	IsTyping       bool     `json:"is_typing"`
	RecipientIDs   []string `json:"recipient_ids"`
}
