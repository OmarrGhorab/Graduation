package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/graduation/chat-service/internal/models"
)

// AuthMiddleware handles JWT authentication
type AuthMiddleware struct {
	jwtSecret string
}

// NewAuthMiddleware creates a new AuthMiddleware
func NewAuthMiddleware(jwtSecret string) *AuthMiddleware {
	return &AuthMiddleware{jwtSecret: jwtSecret}
}

// Authenticate validates JWT token and sets user info in context
func (m *AuthMiddleware) Authenticate() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": fiber.Map{"code": "UNAUTHORIZED", "message": "Authorization header is required"},
			})
		}

		// Extract token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": fiber.Map{"code": "UNAUTHORIZED", "message": "Invalid authorization header format"},
			})
		}
		tokenString := parts[1]

		// Parse and validate token
		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.NewError(fiber.StatusUnauthorized, "Invalid signing method")
			}
			return []byte(m.jwtSecret), nil
		})

		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": fiber.Map{"code": "UNAUTHORIZED", "message": "Invalid or expired token"},
			})
		}

		// Extract claims
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": fiber.Map{"code": "UNAUTHORIZED", "message": "Invalid token claims"},
			})
		}

		// Set user info in context
		c.Locals("user_id", claims["sub"].(string))
		
		// Handle role - could be in "role" field
		if role, ok := claims["role"].(string); ok {
			c.Locals("user_role", models.UserRole(role))
		} else {
			c.Locals("user_role", models.UserRoleStudent) // Default role
		}

		return c.Next()
	}
}

// GetMiddleware returns the authentication middleware function
func (m *AuthMiddleware) GetMiddleware() fiber.Handler {
	return m.Authenticate()
}
