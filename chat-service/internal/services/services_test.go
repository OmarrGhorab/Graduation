package services

import (
	"context"
	"testing"

	"github.com/graduation/chat-service/internal/models"
)

func TestCanCreateGroup(t *testing.T) {
	tests := []struct {
		name     string
		role     models.UserRole
		expected bool
	}{
		{"Instructor can create group", models.UserRoleInstructor, true},
		{"Teacher can create group", models.UserRoleTeacher, true},
		{"Assistant can create group", models.UserRoleAssistant, true},
		{"Student cannot create group", models.UserRoleStudent, false},
		{"Parent cannot create group", models.UserRoleParent, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := canCreateGroup(tt.role)
			if result != tt.expected {
				t.Errorf("canCreateGroup(%v) = %v, want %v", tt.role, result, tt.expected)
			}
		})
	}
}

func TestPermissions_CanModerate(t *testing.T) {
	tests := []struct {
		name     string
		role     models.UserRole
		expected bool
	}{
		{"Instructor can moderate", models.UserRoleInstructor, true},
		{"Teacher can moderate", models.UserRoleTeacher, true},
		{"Assistant can moderate", models.UserRoleAssistant, true},
		{"Student cannot moderate", models.UserRoleStudent, false},
		{"Parent cannot moderate", models.UserRoleParent, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := canModerate(tt.role)
			if result != tt.expected {
				t.Errorf("canModerate(%v) = %v, want %v", tt.role, result, tt.expected)
			}
		})
	}
}

func TestPermissions_CanPin(t *testing.T) {
	tests := []struct {
		name       string
		userRole   models.UserRole
		memberRole models.MemberRole
		expected   bool
	}{
		{"Owner can pin", models.UserRoleStudent, models.MemberRoleOwner, true},
		{"Admin can pin", models.UserRoleStudent, models.MemberRoleAdmin, true},
		{"Member cannot pin", models.UserRoleStudent, models.MemberRoleMember, false},
		{"Instructor can pin (any member role)", models.UserRoleInstructor, models.MemberRoleMember, true},
		{"Teacher can pin (any member role)", models.UserRoleTeacher, models.MemberRoleMember, true},
		{"Assistant can pin (any member role)", models.UserRoleAssistant, models.MemberRoleMember, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := canPin(tt.userRole, tt.memberRole)
			if result != tt.expected {
				t.Errorf("canPin(%v, %v) = %v, want %v", tt.userRole, tt.memberRole, result, tt.expected)
			}
		})
	}
}

func TestTypingService_KeyGeneration(t *testing.T) {
	// Test typing key pattern
	conversationID := "conv-123"
	userID := "user-456"
	expectedKey := "typing:conv-123:user-456"

	key := TypingKeyPrefix + ":" + conversationID + ":" + userID
	if key != expectedKey {
		t.Errorf("Typing key = %v, want %v", key, expectedKey)
	}
}

func TestMediaService_ValidateSize(t *testing.T) {
	svc := NewMediaService("test-cloud", "test-key", "test-secret")

	tests := []struct {
		name      string
		mediaType MediaType
		fileSize  int64
		wantError bool
	}{
		{"Image under limit", MediaTypeImage, 4 * 1024 * 1024, false},
		{"Image at limit", MediaTypeImage, 5 * 1024 * 1024, false},
		{"Image over limit", MediaTypeImage, 6 * 1024 * 1024, true},
		{"Voice under limit", MediaTypeVoice, 14 * 1024 * 1024, false},
		{"Voice at limit", MediaTypeVoice, 15 * 1024 * 1024, false},
		{"Voice over limit", MediaTypeVoice, 16 * 1024 * 1024, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.ValidateMediaSize(tt.mediaType, tt.fileSize)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateMediaSize() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestMediaService_GeneratePresignedURL(t *testing.T) {
	svc := NewMediaService("test-cloud", "test-key", "test-secret")
	ctx := context.Background()
	_ = ctx // Will be used in full implementation

	// Test successful generation
	resp, err := svc.GeneratePresignedURL(MediaTypeImage, "image/jpeg", 1024*1024)
	if err != nil {
		t.Fatalf("GeneratePresignedURL() error = %v", err)
	}

	if resp.UploadURL == "" {
		t.Error("UploadURL should not be empty")
	}
	if resp.Signature == "" {
		t.Error("Signature should not be empty")
	}
	if resp.APIKey != "test-key" {
		t.Errorf("APIKey = %v, want test-key", resp.APIKey)
	}
	if resp.Timestamp == 0 {
		t.Error("Timestamp should not be zero")
	}
}

func TestMediaService_GeneratePresignedURL_SizeValidation(t *testing.T) {
	svc := NewMediaService("test-cloud", "test-key", "test-secret")

	// Test image too large
	_, err := svc.GeneratePresignedURL(MediaTypeImage, "image/jpeg", 10*1024*1024)
	if err == nil {
		t.Error("Expected error for image size > 5MB")
	}

	// Test voice too large
	_, err = svc.GeneratePresignedURL(MediaTypeVoice, "audio/mp3", 16*1024*1024)
	if err == nil {
		t.Error("Expected error for voice size > 15MB")
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"Hello", 10, "Hello"},
		{"Hello World", 5, "He..."},
		{"Short", 100, "Short"},
		{"", 10, ""},
		{"Exactly10!", 10, "Exactly10!"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := truncate(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}
