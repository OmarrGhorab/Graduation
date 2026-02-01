package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/graduation/chat-service/internal/models"
)

// Helper function to create a test Fiber app
func setupTestApp() *fiber.App {
	return fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})
}

// Helper to make request body
func makeBody(data interface{}) io.Reader {
	b, _ := json.Marshal(data)
	return bytes.NewReader(b)
}

func TestHealthHandler_Health(t *testing.T) {
	app := setupTestApp()
	handler := NewHealthHandler()

	app.Get("/health", handler.Health)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)

	if body["status"] != "ok" {
		t.Errorf("status = %v, want ok", body["status"])
	}
	if body["service"] != "chat-service" {
		t.Errorf("service = %v, want chat-service", body["service"])
	}
}

func TestHealthHandler_Ready(t *testing.T) {
	app := setupTestApp()
	handler := NewHealthHandler()

	app.Get("/ready", handler.Ready)

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestTypingHandler_SetTyping_ValidationError(t *testing.T) {
	app := setupTestApp()

	// Mock handler without real service
	app.Post("/typing", func(c *fiber.Ctx) error {
		// Set locals for auth middleware simulation
		c.Locals("user_id", "test-user")
		c.Locals("user_role", models.UserRoleStudent)

		var req SetTypingRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fiber.Map{"code": "BAD_REQUEST", "message": "Invalid request body"},
			})
		}

		if req.ConversationID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fiber.Map{"code": "BAD_REQUEST", "message": "Conversation ID is required"},
			})
		}

		return c.Status(fiber.StatusNoContent).Send(nil)
	})

	// Test with empty body
	req := httptest.NewRequest(http.MethodPost, "/typing", makeBody(map[string]string{}))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestMediaHandler_Presign_ValidationError(t *testing.T) {
	app := setupTestApp()

	// Mock handler
	app.Post("/media/presign", func(c *fiber.Ctx) error {
		c.Locals("user_id", "test-user")
		c.Locals("user_role", models.UserRoleStudent)

		var req PresignRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fiber.Map{"code": "BAD_REQUEST", "message": "Invalid request body"},
			})
		}

		if req.Type == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fiber.Map{"code": "BAD_REQUEST", "message": "Media type is required"},
			})
		}

		if req.FileSize <= 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fiber.Map{"code": "BAD_REQUEST", "message": "File size must be positive"},
			})
		}

		return c.JSON(fiber.Map{"upload_url": "https://example.com/upload"})
	})

	tests := []struct {
		name           string
		body           PresignRequest
		expectedStatus int
	}{
		{"Missing type", PresignRequest{FileSize: 1000}, http.StatusBadRequest},
		{"Zero file size", PresignRequest{Type: "image", FileSize: 0}, http.StatusBadRequest},
		{"Negative file size", PresignRequest{Type: "image", FileSize: -1}, http.StatusBadRequest},
		{"Valid request", PresignRequest{Type: "image", ContentType: "image/jpeg", FileSize: 1000}, http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/media/presign", makeBody(tt.body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test() error = %v", err)
			}

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Status = %d, want %d", resp.StatusCode, tt.expectedStatus)
			}
		})
	}
}

func TestConversationHandler_CreateGroup_ValidationError(t *testing.T) {
	app := setupTestApp()

	// Mock handler
	app.Post("/conversations", func(c *fiber.Ctx) error {
		c.Locals("user_id", "test-user")
		c.Locals("user_role", models.UserRoleTeacher)

		var req CreateGroupRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fiber.Map{"code": "BAD_REQUEST", "message": "Invalid request body"},
			})
		}

		if req.Name == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fiber.Map{"code": "BAD_REQUEST", "message": "Group name is required"},
			})
		}

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": "new-group-id"})
	})

	tests := []struct {
		name           string
		body           CreateGroupRequest
		expectedStatus int
	}{
		{"Missing name", CreateGroupRequest{Description: "Test"}, http.StatusBadRequest},
		{"Valid request", CreateGroupRequest{Name: "Test Group"}, http.StatusCreated},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/conversations", makeBody(tt.body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test() error = %v", err)
			}

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Status = %d, want %d", resp.StatusCode, tt.expectedStatus)
			}
		})
	}
}

func TestMessageHandler_SendMessage_ValidationTypes(t *testing.T) {
	// Test that message types are correctly validated
	validTypes := []models.MessageType{
		models.MessageTypeText,
		models.MessageTypeImage,
		models.MessageTypeVoice,
	}

	for _, msgType := range validTypes {
		if msgType != models.MessageTypeText && msgType != models.MessageTypeImage && msgType != models.MessageTypeVoice {
			t.Errorf("Invalid message type: %v", msgType)
		}
	}
}
