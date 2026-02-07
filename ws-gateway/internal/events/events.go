package events

import "time"

const (
	TopicMessageCreated   = "chat.message.created"
	TopicTyping           = "chat.typing"
	TopicReadReceipt      = "chat.read.receipt"
	TopicUserPresence     = "chat.user.presence"
	TopicConversationRead = "chat.conversation.read"
)

type MessageCreatedEvent struct {
	ID                    string         `json:"id"`
	ConversationID        string         `json:"conversation_id"`
	SenderID              string         `json:"sender_id"`
	Content               string         `json:"content"`
	Type                  string         `json:"type"`
	MediaURLs             []string       `json:"media_urls,omitempty"`
	CreatedAt             time.Time      `json:"created_at"`
	RecipientIDs          []string       `json:"recipient_ids"`
	RecipientUnreadCounts map[string]int `json:"recipient_unread_counts,omitempty"`
}

type TypingEvent struct {
	ConversationID string   `json:"conversation_id"`
	UserID         string   `json:"user_id"`
	UserName       string   `json:"user_name,omitempty"`
	IsTyping       bool     `json:"is_typing"`
	RecipientIDs   []string `json:"recipient_ids"`
}

type UserPresenceEvent struct {
	UserID    string `json:"user_id"`
	IsOnline  bool   `json:"is_online"`
	Timestamp string `json:"timestamp"`
}

type ConversationReadEvent struct {
	ConversationID string `json:"conversation_id"`
	UserID         string `json:"user_id"`
	UnreadCount    int    `json:"unread_count"`
}

// Internal Envelope for Redis Pub/Sub
type RedisEnvelope struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
	Targets []string    `json:"targets"` // Recipient UserIDs
}
