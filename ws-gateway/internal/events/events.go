package events

import "time"

const (
	TopicMessageCreated = "chat.message.created"
	TopicTyping         = "chat.typing"
	TopicReadReceipt    = "chat.read.receipt"
)

type MessageCreatedEvent struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	SenderID       string    `json:"sender_id"`
	Content        string    `json:"content"`
	Type           string    `json:"type"`
	MediaURLs      []string  `json:"media_urls,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	RecipientIDs   []string  `json:"recipient_ids"`
}

type TypingEvent struct {
	ConversationID string   `json:"conversation_id"`
	UserID         string   `json:"user_id"`
	IsTyping       bool     `json:"is_typing"`
	RecipientIDs   []string `json:"recipient_ids"`
}

// Internal Envelope for Redis Pub/Sub
type RedisEnvelope struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
	Targets []string    `json:"targets"` // Recipient UserIDs
}
