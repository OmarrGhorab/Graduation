package handlers

import (
	"fmt"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/graduation/ws-gateway/internal/config"
	wsCore "github.com/graduation/ws-gateway/internal/websocket"
)

func WebSocketHandler(manager *wsCore.Manager, cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 1. Auth Logic (simplified for brevity, should use middleware ideally or inline here)
		tokenString := c.Query("token")
		if tokenString == "" {
			return c.Status(401).SendString("Missing token")
		}

		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected method")
			}
			return []byte(cfg.JwtSecret), nil
		})

		if err != nil || !token.Valid {
			return c.Status(401).SendString("Invalid token")
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return c.Status(401).SendString("Invalid claims")
		}

		userID := claims["sub"].(string)
		c.Locals("user_id", userID)

		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	}
}

func WebSocketConnection(manager *wsCore.Manager) func(*websocket.Conn) {
	return func(c *websocket.Conn) {
		userID := c.Locals("user_id").(string)

		client := &wsCore.Client{
			Manager: manager,
			UserID:  userID,
			Conn:    c,
			Send:    make(chan []byte, 256), // Buffer
		}

		manager.Register <- client

		// Start Pumps
		go client.WritePump()
		client.ReadPump()
	}
}
