package services

import (
	"context"
	"testing"
	"time"
)

func TestTypingService_Constants(t *testing.T) {
	if TypingKeyPrefix != "typing" {
		t.Errorf("TypingKeyPrefix = %v, want typing", TypingKeyPrefix)
	}

	if TypingTTL != 3*time.Second {
		t.Errorf("TypingTTL = %v, want 3s", TypingTTL)
	}
}

func TestPollService_Constants(t *testing.T) {
	if PollKeyPrefix != "poll" {
		t.Errorf("PollKeyPrefix = %v, want poll", PollKeyPrefix)
	}
}

func TestPollResponse_EmptyMessages(t *testing.T) {
	resp := &PollResponse{
		Messages: nil,
		HasMore:  false,
	}

	if len(resp.Messages) != 0 {
		t.Error("Expected empty messages slice")
	}

	if resp.HasMore {
		t.Error("HasMore should be false")
	}
}

// Test GetTypingUsers without Redis (unit test)
func TestTypingService_NewTypingService(t *testing.T) {
	// Without redis client (nil is acceptable for unit test setup)
	svc := NewTypingService(nil)
	if svc == nil {
		t.Error("NewTypingService returned nil")
	}
}

func TestTypingUser_Struct(t *testing.T) {
	user := TypingUser{
		UserID:   "user-123",
		UserRole: "TEACHER",
	}

	if user.UserID != "user-123" {
		t.Errorf("UserID = %v, want user-123", user.UserID)
	}
}

func TestPollService_NewPollService(t *testing.T) {
	timeout := 30 * time.Second
	interval := 500 * time.Millisecond

	svc := NewPollService(nil, nil, timeout, interval)
	if svc == nil {
		t.Error("NewPollService returned nil")
	}
	if svc.pollTimeout != timeout {
		t.Errorf("pollTimeout = %v, want %v", svc.pollTimeout, timeout)
	}
	if svc.pollInterval != interval {
		t.Errorf("pollInterval = %v, want %v", svc.pollInterval, interval)
	}
}

// Test context cancellation handling
func TestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	select {
	case <-ctx.Done():
		// Expected
	default:
		t.Error("Context should be cancelled")
	}
}
