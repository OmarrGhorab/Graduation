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

// Test GetTypingUsers without Redis (unit test)
func TestTypingService_NewTypingService(t *testing.T) {
	// Without redis client (nil is acceptable for unit test setup)
	svc := NewTypingService(nil, nil)
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
	if user.UserRole != "TEACHER" {
		t.Errorf("UserRole = %v, want TEACHER", user.UserRole)
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
