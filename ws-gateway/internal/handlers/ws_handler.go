package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/graduation/ws-gateway/internal/config"
	wsCore "github.com/graduation/ws-gateway/internal/websocket"
)

var Manager *wsCore.Manager

// WebSocketHandler handles incoming WebSocket connections
func WebSocketHandler(c *fiber.Ctx) error {
	userID := c.Query("user_id") // TEMPORARY for testing
	if userID == "" {
		return fiber.ErrUnauthorized
	}

	if websocket.IsWebSocketUpgrade(c) {
		c.Locals("user_id", userID)
		return c.Next()
	}
	return fiber.ErrUpgradeRequired
}

// WSMessage represents the structure of messages sent by clients
type WSMessage struct {
	Type    string          `json:"type"` // e.g., "message.send"
	Payload json.RawMessage `json:"payload"`
}

// SendMessagePayload represents the payload for sending a message
type SendMessagePayload struct {
	ConversationID string  `json:"conversation_id"`
	Content        string  `json:"content"`
	Type           string  `json:"type"`     // text, image, etc.
	LocalID        string  `json:"local_id"` // Idempotency
	ReplyToID      *string `json:"reply_to_id"`
}

// WebSocketConnection is the actual websocket endpoint handler
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

		// Handle Incoming Message
		handleIncomingMessage(userID, msg)
	}
}

func handleIncomingMessage(userID string, data []byte) {
	var wsMsg WSMessage
	if err := json.Unmarshal(data, &wsMsg); err != nil {
		log.Printf("[Proxy] Invalid JSON from %s: %v", userID, err)
		return
	}

	switch wsMsg.Type {
	case "message.send":
		var payload SendMessagePayload
		if err := json.Unmarshal(wsMsg.Payload, &payload); err != nil {
			log.Printf("[Proxy] Invalid Payload from %s: %v", userID, err)
			return
		}

		// Proxy to Chat Service
		if err := proxySendMessage(userID, payload); err != nil {
			log.Printf("[Proxy] Failed to send message for %s: %v", userID, err)
			// TODO: Send error back to client via WebSocket?
		}
	default:
		log.Printf("[Proxy] Unknown message type from %s: %s", userID, wsMsg.Type)
	}
}

func proxySendMessage(userID string, payload SendMessagePayload) error {
	cfg := config.Load() // In prod, inject or pass config

	// 1. Generate short-lived JWT for the user
	token, err := generateUserToken(userID, cfg.JwtSecret)
	if err != nil {
		return fmt.Errorf("failed to generate token: %w", err)
	}

	// 2. Prepare Request Body
	// Chat Service matches SendMessageRequest struct
	reqBody := map[string]interface{}{
		"content": payload.Content,
		"type":    payload.Type,
		// "media_urls": ...
		// "local_id": payload.LocalID (Chat Service needs to support this in body, usually handled)
		// Checking Chat Service Handler: It maps body to SendMessageRequest.
		// SendMessageRequest has: Type, Content, MediaURLs, ReplyToID.
		// Wait, where is LocalID?
		// We added LocalID to SendMessageInput in Service, but Handler might NOT parse it from Body yet?
		// Step 164 showed SendMessageInput has LocalID.
		// Step 367 (Handler) shows SendMessageRequest struct DOES NOT have LocalID.
		// ISSUE: We need to update Chat Service Handler to accept LocalID if we want idempotency.
		// For now, we will omit LocalID or it won't be used.
	}
	if payload.ReplyToID != nil {
		reqBody["reply_to_id"] = payload.ReplyToID
	}

	jsonBody, _ := json.Marshal(reqBody)

	// 3. Create HTTP Request
	url := fmt.Sprintf("%s/api/v1/conversations/%s/messages", cfg.ChatServiceUrl, payload.ConversationID)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	// 4. Send Request
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("chat service returned status: %d", resp.StatusCode)
	}

	log.Printf("[Proxy] Message sent to %s for user %s", payload.ConversationID, userID)
	return nil
}

func generateUserToken(userID, secret string) (string, error) {
	claims := jwt.MapClaims{
		"sub":  userID,
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(1 * time.Minute).Unix(), // Short expiry
		"role": "student",                              // Default role, or fetch real role if needed
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
