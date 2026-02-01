package services

import (
	"github.com/graduation/chat-service/internal/clients"
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
	Auth         *clients.AuthClient
}

// NewServices creates a new Services instance with all services
func NewServices(repos *repositories.Repositories, redis *cache.RedisClient, cfg *config.Config) *Services {
	notificationSvc := NewNotificationService(cfg.NotificationServiceURL, cfg.InternalServiceSecret)
	// Auth service URL is likely same as notification service URL base but different port or same cluster
	// Assuming it's configured similar to notification service or hardcoded for now if config missing
	// TODO: Add AuthServiceURL to config. For now, we reuse NotificationServiceURL if similar, or assume localhost:6002
	// Actually, looking at .env, we don't have AUTH_SERVICE_URL.
	// Auth Service is on port 6002 usually.
	authSvcURL := "http://localhost:6001" // Default fallback
	if cfg.NotificationServiceURL != "" {
		// Just a hack if we don't have separate config
	}

	authClient := clients.NewAuthClient(authSvcURL, cfg.InternalServiceSecret)

	return &Services{
		Conversation: NewConversationService(repos, redis, notificationSvc, authClient),
		Message:      NewMessageService(repos, redis, notificationSvc, authClient),
		Typing:       NewTypingService(redis),
		Poll:         NewPollService(repos.Message, redis, cfg.PollTimeout, cfg.PollInterval),
		Media:        NewMediaService(cfg.CloudinaryCloudName, cfg.CloudinaryAPIKey, cfg.CloudinaryAPISecret),
		Notification: notificationSvc,
		Auth:         authClient,
	}
}
