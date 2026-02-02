package events

const (
	// TopicMessageCreated is the topic for new messages.
	// Partitioned by: conversation_id
	TopicMessageCreated = "chat.message.created"

	// TopicMessageUpdated is the topic for message edits, deletes, and pins.
	// Partitioned by: conversation_id
	TopicMessageUpdated = "chat.message.updated"

	// TopicReadReceipt is the topic for read receipts.
	// Partitioned by: conversation_id
	TopicReadReceipt = "chat.read.receipt"

	// TopicTyping is the topic for typing indicators.
	// Partitioned by: conversation_id
	TopicTyping = "chat.typing"
)
