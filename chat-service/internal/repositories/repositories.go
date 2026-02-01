package repositories

import (
	"gorm.io/gorm"
)

// Repositories holds all repository instances
type Repositories struct {
	Conversation *ConversationRepository
	Message      *MessageRepository
	Member       *MemberRepository
	DeviceToken  *DeviceTokenRepository
}

// NewRepositories creates a new Repositories instance with all repositories
func NewRepositories(db *gorm.DB) *Repositories {
	return &Repositories{
		Conversation: NewConversationRepository(db),
		Message:      NewMessageRepository(db),
		Member:       NewMemberRepository(db),
		DeviceToken:  NewDeviceTokenRepository(db),
	}
}
