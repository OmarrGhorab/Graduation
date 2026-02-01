package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/graduation/chat-service/internal/models"
)

func setupTestApp() *fiber.App {
	return fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})
}

func createTestToken(secret string, claims jwt.MapClaims) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(secret))
	return tokenString
}

func TestAuthMiddleware_MissingAuthHeader(t *testing.T) {
	app := setupTestApp()
	middleware := NewAuthMiddleware("test-secret")

	app.Get("/protected", middleware.Authenticate(), func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "success"})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestAuthMiddleware_InvalidHeaderFormat(t *testing.T) {
	app := setupTestApp()
	middleware := NewAuthMiddleware("test-secret")

	app.Get("/protected", middleware.Authenticate(), func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "success"})
	})

	// Missing "Bearer" prefix
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "some-token")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	app := setupTestApp()
	middleware := NewAuthMiddleware("test-secret")

	app.Get("/protected", middleware.Authenticate(), func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "success"})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestAuthMiddleware_WrongSecret(t *testing.T) {
	app := setupTestApp()
	middleware := NewAuthMiddleware("correct-secret")

	app.Get("/protected", middleware.Authenticate(), func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "success"})
	})

	// Create token with different secret
	token := createTestToken("wrong-secret", jwt.MapClaims{
		"sub":  "user-123",
		"role": "STUDENT",
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	secret := "test-secret"
	app := setupTestApp()
	middleware := NewAuthMiddleware(secret)

	var capturedUserID string
	var capturedRole string

	app.Get("/protected", middleware.Authenticate(), func(c *fiber.Ctx) error {
		capturedUserID = c.Locals("user_id").(string)
		capturedRole = string(c.Locals("user_role").(models.UserRole))
		return c.JSON(fiber.Map{"message": "success"})
	})

	token := createTestToken(secret, jwt.MapClaims{
		"sub":  "user-123",
		"role": "TEACHER",
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	if capturedUserID != "user-123" {
		t.Errorf("user_id = %v, want user-123", capturedUserID)
	}

	if capturedRole != "TEACHER" {
		t.Errorf("user_role = %v, want TEACHER", capturedRole)
	}
}

func TestAuthMiddleware_TokenWithoutRole(t *testing.T) {
	secret := "test-secret"
	app := setupTestApp()
	middleware := NewAuthMiddleware(secret)

	app.Get("/protected", middleware.Authenticate(), func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "success"})
	})

	// Token without role claim
	token := createTestToken(secret, jwt.MapClaims{
		"sub": "user-123",
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}

	// Should still succeed, default role applied
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestNewAuthMiddleware(t *testing.T) {
	middleware := NewAuthMiddleware("test-secret")
	if middleware == nil {
		t.Error("NewAuthMiddleware() returned nil")
	}
	if middleware.jwtSecret != "test-secret" {
		t.Errorf("jwtSecret = %v, want test-secret", middleware.jwtSecret)
	}
}

func TestGetMiddleware(t *testing.T) {
	middleware := NewAuthMiddleware("test-secret")
	handler := middleware.GetMiddleware()
	if handler == nil {
		t.Error("GetMiddleware() returned nil")
	}
}
