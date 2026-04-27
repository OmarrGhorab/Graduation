package middleware

import (
	"fmt"
	"strings"

	"github.com/OmarrGhorab/payment-service/internal/infrastructure/authclient"
	"github.com/gofiber/fiber/v2"
)

func Authenticate(authClient *authclient.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		path := c.Path()
		fmt.Printf("[Auth Debug] Request Path: %s\n", path)
		
		// Skip authentication for Paymob webhook and public status checks
		if strings.Contains(path, "/webhook/paymob") || strings.EqualFold(path, "/api/v1/payments/status") || strings.HasSuffix(path, "/status") {
			return c.Next()
		}

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
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "Invalid or expired token",
			})
		}

		if !resp.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "Unauthorized",
			})
		}

		c.Locals("userId", resp.UserID)
		c.Locals("userRole", resp.Role)

		return c.Next()
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
