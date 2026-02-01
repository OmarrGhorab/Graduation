package services

import (
	"github.com/graduation/chat-service/internal/config"
	"github.com/graduation/chat-service/internal/repositories"
	"github.com/graduation/chat-service/pkg/cache"
)

// Services holds all service instances
type Services struct {
	Conversation *ConversationService
	Message      *MessageService
	Typing       *TypingService
	Poll         *PollService
	Media        *MediaService
	Notification *NotificationService
}

// NewServices creates a new Services instance with all services
func NewServices(repos *repositories.Repositories, redis *cache.RedisClient, cfg *config.Config) *Services {
	notificationSvc := NewNotificationService(cfg.NotificationServiceURL, cfg.InternalServiceSecret)
	
	return &Services{
		Conversation: NewConversationService(repos, redis, notificationSvc),
		Message:      NewMessageService(repos, redis, notificationSvc),
		Typing:       NewTypingService(redis),
		Poll:         NewPollService(repos.Message, redis, cfg.PollTimeout, cfg.PollInterval),
		Media:        NewMediaService(cfg.CloudinaryCloudName, cfg.CloudinaryAPIKey, cfg.CloudinaryAPISecret),
		Notification: notificationSvc,
	}
}
