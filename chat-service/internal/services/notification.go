package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/graduation/chat-service/internal/models"
)

// NotificationService handles push notification logic via notification-service
type NotificationService struct {
	serviceURL     string
	internalSecret string
	httpClient     *http.Client
}

// NewNotificationService creates a new NotificationService
func NewNotificationService(serviceURL, internalSecret string) *NotificationService {
	return &NotificationService{
		serviceURL:     serviceURL,
		internalSecret: internalSecret,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ChatNotificationPayload is the payload for a chat notification
type ChatNotificationPayload struct {
	UserID           string `json:"userId"`
	Type             string `json:"type"`
	SenderName       string `json:"senderName"`
	SenderRole       string `json:"senderRole"`
	ConversationID   string `json:"conversationId"`
	ConversationName string `json:"conversationName"`
	MessagePreview   string `json:"messagePreview"`
	MessageType      string `json:"messageType"`
}

// SendChatNotification sends a chat notification to the notification-service
func (s *NotificationService) SendChatNotification(ctx context.Context, message *models.Message, conversation *models.Conversation, recipientIDs []string) error {
	for _, recipientID := range recipientIDs {
		payload := ChatNotificationPayload{
			UserID:           recipientID,
			Type:             "CHAT_MESSAGE",
			SenderName:       message.SenderID, // TODO: Get actual name from auth service
			SenderRole:       string(message.SenderRole),
			ConversationID:   conversation.ID,
			ConversationName: getConversationName(conversation),
			MessagePreview:   truncate(message.Content, 100),
			MessageType:      string(message.Type),
		}

		go s.sendNotification(ctx, payload)
	}
	return nil
}

// sendNotification sends a single notification to the notification-service
func (s *NotificationService) sendNotification(ctx context.Context, payload ChatNotificationPayload) {
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("[Notification] Error marshaling payload: %v\n", err)
		return
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.serviceURL+"/api/v1/notifications/publish", bytes.NewBuffer(jsonBody))
	if err != nil {
		fmt.Printf("[Notification] Error creating request: %v\n", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-internal-service-secret", s.internalSecret)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		fmt.Printf("[Notification] Error sending request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("[Notification] Unexpected status code: %d\n", resp.StatusCode)
	}
}

func getConversationName(conv *models.Conversation) string {
	if conv.Name != "" {
		return conv.Name
	}
	if conv.Type == models.ConversationTypeDirect {
		return "Direct Message"
	}
	return "Group Chat"
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
