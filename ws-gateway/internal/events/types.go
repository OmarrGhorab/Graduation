package events

import "time"

// MessageCreatedEvent represents a new message event
type MessageCreatedEvent struct {
	ID             string    `json:"id"`
	LocalID        string    `json:"local_id"`
	ConversationID string    `json:"conversation_id"`
	SenderID       string    `json:"sender_id"`
	Content        string    `json:"content"`
	Type           string    `json:"type"` // text, image, voice
	MediaURLs      []string  `json:"media_urls,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// MessageUpdatedEvent represents a message update (edit, delete, pin)
type MessageUpdatedEvent struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	Action         string    `json:"action"` // edit, delete, pin
	Content        string    `json:"content,omitempty"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ReadReceiptEvent represents a read receipt
type ReadReceiptEvent struct {
	UserID         string    `json:"user_id"`
	ConversationID string    `json:"conversation_id"`
	MessageID      string    `json:"message_id"`
	ReadAt         time.Time `json:"read_at"`
}

// TypingEvent represents a typing indicator
type TypingEvent struct {
	UserID         string `json:"user_id"`
	ConversationID string `json:"conversation_id"`
	IsTyping       bool   `json:"is_typing"`
}
