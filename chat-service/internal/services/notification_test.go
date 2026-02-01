package services

import (
	"context"
	"testing"

	"github.com/graduation/chat-service/internal/models"
)

func TestNotificationService_NewNotificationService(t *testing.T) {
	svc := NewNotificationService("http://localhost:6003", "test-secret")
	if svc == nil {
		t.Error("NewNotificationService returned nil")
	}
	if svc.serviceURL != "http://localhost:6003" {
		t.Errorf("serviceURL = %v, want http://localhost:6003", svc.serviceURL)
	}
	if svc.internalSecret != "test-secret" {
		t.Errorf("internalSecret = %v, want test-secret", svc.internalSecret)
	}
	if svc.httpClient == nil {
		t.Error("httpClient should not be nil")
	}
}

func TestChatNotificationPayload_Structure(t *testing.T) {
	payload := ChatNotificationPayload{
		UserID:           "user-123",
		Type:             "CHAT_MESSAGE",
		SenderName:       "John",
		SenderRole:       "TEACHER",
		ConversationID:   "conv-123",
		ConversationName: "Test Group",
		MessagePreview:   "Hello world",
		MessageType:      "text",
	}

	if payload.UserID != "user-123" {
		t.Errorf("UserID = %v, want user-123", payload.UserID)
	}
	if payload.Type != "CHAT_MESSAGE" {
		t.Errorf("Type = %v, want CHAT_MESSAGE", payload.Type)
	}
	if payload.MessageType != "text" {
		t.Errorf("MessageType = %v, want text", payload.MessageType)
	}
	if payload.SenderName != "John" {
		t.Errorf("SenderName = %v, want John", payload.SenderName)
	}
	if payload.SenderRole != "TEACHER" {
		t.Errorf("SenderRole = %v, want TEACHER", payload.SenderRole)
	}
	if payload.ConversationID != "conv-123" {
		t.Errorf("ConversationID = %v, want conv-123", payload.ConversationID)
	}
	if payload.ConversationName != "Test Group" {
		t.Errorf("ConversationName = %v, want Test Group", payload.ConversationName)
	}
	if payload.MessagePreview != "Hello world" {
		t.Errorf("MessagePreview = %v, want Hello world", payload.MessagePreview)
	}
}

func TestGetConversationName(t *testing.T) {
	tests := []struct {
		name     string
		conv     *models.Conversation
		expected string
	}{
		{
			name:     "Group with name",
			conv:     &models.Conversation{Name: "Study Group", Type: models.ConversationTypeGroup},
			expected: "Study Group",
		},
		{
			name:     "Direct chat without name",
			conv:     &models.Conversation{Name: "", Type: models.ConversationTypeDirect},
			expected: "Direct Message",
		},
		{
			name:     "Group without name",
			conv:     &models.Conversation{Name: "", Type: models.ConversationTypeGroup},
			expected: "Group Chat",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getConversationName(tt.conv)
			if result != tt.expected {
				t.Errorf("getConversationName() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSendChatNotification_EmptyRecipients(t *testing.T) {
	svc := NewNotificationService("http://localhost:6003", "test-secret")
	ctx := context.Background()

	msg := &models.Message{
		ID:      "msg-1",
		Content: "Hello",
	}
	conv := &models.Conversation{
		ID:   "conv-1",
		Name: "Test",
	}

	// Should not panic with empty recipients
	err := svc.SendChatNotification(ctx, msg, conv, []string{})
	if err != nil {
		t.Errorf("SendChatNotification() error = %v", err)
	}
}
