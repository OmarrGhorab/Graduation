package middleware

import (
	"fmt"
	"strings"

	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/authclient"
	"github.com/gofiber/fiber/v2"
)

// Authenticate validates the JWT token with the auth service
func Authenticate(authClient *authclient.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "Missing authorization header",
			})
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "Invalid authorization header format",
			})
		}

		token := parts[1]

		resp, err := authClient.ValidateToken(c.Context(), token)
		if err != nil {
			fmt.Printf("[Auth Debug] Token validation failed: %v\n", err)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "Invalid or expired token",
			})
		}
		
		fmt.Printf("[Auth Debug] Validation response: Valid=%v, UserID=%s\n", resp.Valid, resp.UserID)

		if !resp.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "Unauthorized",
			})
		}

		// Store in locals for handlers to use
		c.Locals("userId", resp.UserID)
		c.Locals("userRole", resp.Role)

		return c.Next()
	}
}

// RequireRole checks if the user has one of the allowed roles
func RequireRole(allowedRoles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userRole := c.Locals("userRole")
		if userRole == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "Unauthorized",
			})
		}

		roleStr, ok := userRole.(string)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"error":   "Invalid user role",
			})
		}

		for _, role := range allowedRoles {
			if roleStr == role {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error":   "You do not have permission to perform this action",
		})
	}
}
// InternalOnly ensures the request has a valid internal service secret
func InternalOnly(secret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		providedSecret := c.Get("x-internal-service-secret")
		if providedSecret == "" || providedSecret != secret {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"error":   "Forbidden: Internal use only",
			})
		}
		return c.Next()
	}
}
