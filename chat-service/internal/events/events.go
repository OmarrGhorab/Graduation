package events

import "time"

// Kafka Topics
const (
	TopicMessageCreated   = "chat.message.created"
	TopicTyping           = "chat.typing"
	TopicReadReceipt      = "chat.read.receipt"
	TopicConversationRead = "chat.conversation.read"
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

	// Enriched Sender Info (New)
	SenderName  string `json:"sender_name,omitempty"`
	SenderImage string `json:"sender_image,omitempty"`

	// Routing Info (CRITICAL for Gateway)
	RecipientIDs          []string       `json:"recipient_ids"`
	RecipientUnreadCounts map[string]int `json:"recipient_unread_counts,omitempty"`
}

// TypingEvent is published when a user starts/stops typing
// Key: ConversationID
type TypingEvent struct {
	ConversationID string   `json:"conversation_id"`
	UserID         string   `json:"user_id"`
	UserName       string   `json:"user_name,omitempty"`
	UserImage      string   `json:"user_image,omitempty"`
	IsTyping       bool     `json:"is_typing"`
	RecipientIDs   []string `json:"recipient_ids"`
}

type ConversationReadEvent struct {
	ConversationID string `json:"conversation_id"`
	UserID         string `json:"user_id"`
	UnreadCount    int    `json:"unread_count"` // Should be 0
}
