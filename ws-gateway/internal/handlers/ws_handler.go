package handlers

import (
	"log"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	wsCore "github.com/graduation/ws-gateway/internal/websocket"
)

var Manager *wsCore.Manager

// WebSocketHandler handles incoming WebSocket connections
func WebSocketHandler(c *fiber.Ctx) error {
	// Authentication (e.g., via Query Param ?token=...)
	// For MVP: assume middleware has validated or we extract user_id from query.
	userID := c.Query("user_id") // TEMPORARY for testing

	if userID == "" {
		// Try to parse token if provided, else unauthorized
		// In production: userID, err := ParseToken(c.Query("token"))
		return fiber.ErrUnauthorized
	}

	if websocket.IsWebSocketUpgrade(c) {
		c.Locals("user_id", userID)
		return c.Next()
	}
	return fiber.ErrUpgradeRequired
}

// WebSocketConnection is the actual websocket endpoint handler (passed to fiber-websocket)
func WebSocketConnection(c *websocket.Conn) {
	userID := c.Locals("user_id").(string)

	client := &wsCore.Client{
		UserID: userID,
		Conn:   c,
		Send:   make(chan []byte, 256),
	}

	Manager.Register <- client

	// Write Pump
	go func() {
		defer func() {
			Manager.Unregister <- client
			c.Close()
		}()
		for {
			message, ok := <-client.Send
			if !ok {
				c.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		}
	}()

	// Read Pump (Client -> Gateway)
	// Gateway should forward "message.send" to Chat Service via HTTP
	// For now, we just log.
	for {
		// Set basic read deadline logic
		c.SetReadDeadline(time.Now().Add(60 * time.Second))
		c.SetPongHandler(func(string) error {
			c.SetReadDeadline(time.Now().Add(60 * time.Second))
			return nil
		})

		_, msg, err := c.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WS Error user=%s: %v", userID, err)
			}
			break
		}

		// Placeholder for proxy logic:
		log.Printf("Msg from %s: %s", userID, string(msg))
		// TODO: Validate message, Http Post to Chat Service
	}
}
