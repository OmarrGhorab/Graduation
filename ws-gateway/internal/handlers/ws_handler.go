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
	// 1. Get Token from Query or Header
	tokenString := c.Query("token")
	if tokenString == "" {
		authHeader := c.Get("Authorization")
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			tokenString = authHeader[7:]
		}
	}

	if tokenString == "" {
		log.Println("[Auth] Missing token")
		return c.Status(fiber.StatusUnauthorized).SendString("Missing token")
	}

	// 2. Parse and Validate Token
	cfg := config.Load()
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(cfg.JwtSecret), nil
	})

	if err != nil || !token.Valid {
		log.Printf("[Auth] Invalid token: %v", err)
		return c.Status(fiber.StatusForbidden).SendString("Invalid token")
	}

	// 3. Extract User ID
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return c.Status(fiber.StatusForbidden).SendString("Invalid claims")
	}
	userID, ok := claims["sub"].(string)
	if !ok || userID == "" {
		// Fallback for some JWT structures (id vs sub)
		if id, ok := claims["id"].(string); ok {
			userID = id
		} else {
			return c.Status(fiber.StatusForbidden).SendString("No user ID in token")
		}
	}

	// 4. Pass UserID and Role to Locals for Upgrade
	c.Locals("user_id", userID)
	if role, ok := claims["role"].(string); ok {
		c.Locals("user_role", role)
	} else {
		c.Locals("user_role", "student")
	}

	if websocket.IsWebSocketUpgrade(c) {
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
	userRole := c.Locals("user_role").(string)

	client := &wsCore.Client{
		UserID:   userID,
		UserRole: userRole,
		Conn:     c,
		Send:     make(chan []byte, 256),
	}

	Manager.Register <- client

	// Write Pump
	go func() {
		ticker := time.NewTicker(50 * time.Second)
		defer func() {
			ticker.Stop()
			Manager.Unregister <- client
			c.Close()
		}()
		for {
			select {
			case message, ok := <-client.Send:
				if !ok {
					c.WriteMessage(websocket.CloseMessage, []byte{})
					return
				}
				if err := c.WriteMessage(websocket.TextMessage, message); err != nil {
					return
				}
			case <-ticker.C:
				if err := c.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
					return
				}
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
			} else {
				log.Printf("WS Closed for user=%s: %v", userID, err)
			}
			break
		}

		// Handle Incoming Message
		handleIncomingMessage(client, msg)
	}
}

func handleIncomingMessage(client *wsCore.Client, data []byte) {
	userID := client.UserID
	userRole := client.UserRole
	var wsMsg WSMessage
	if err := json.Unmarshal(data, &wsMsg); err != nil {
		log.Printf("[Proxy] Invalid JSON from %s: %v", userID, err)
		return
	}

	log.Printf("[Proxy] Received message type '%s' from user %s", wsMsg.Type, userID)

	switch wsMsg.Type {
	case "message.send":
		var payload SendMessagePayload
		if err := json.Unmarshal(wsMsg.Payload, &payload); err != nil {
			log.Printf("[Proxy] Invalid Payload from %s: %v", userID, err)
			return
		}

		// Proxy to Chat Service
		if err := proxySendMessage(userID, userRole, payload); err != nil {
			log.Printf("[Proxy] Failed to send message for %s: %v", userID, err)
		}
	case "typing.start":
		var payload struct {
			ConversationID string `json:"conversation_id"`
		}
		if err := json.Unmarshal(wsMsg.Payload, &payload); err != nil {
			log.Printf("[Proxy] Invalid Typing Payload from %s: %v", userID, err)
			return
		}

		log.Printf("[Proxy] Forwarding typing.start from %s for conversation %s", userID, payload.ConversationID)
		if err := proxyTyping(userID, userRole, payload.ConversationID); err != nil {
			log.Printf("[Proxy] Failed to send typing for %s: %v", userID, err)
		}
	default:
		log.Printf("[Proxy] Unknown message type from %s: %s", userID, wsMsg.Type)
	}
}

func proxySendMessage(userID string, userRole string, payload SendMessagePayload) error {
	cfg := config.Load() // In prod, inject or pass config

	// 1. Generate short-lived JWT for the user
	token, err := generateUserToken(userID, userRole, cfg.JwtSecret)
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

func proxyTyping(userID string, userRole string, conversationID string) error {
	cfg := config.Load()
	token, err := generateUserToken(userID, userRole, cfg.JwtSecret)
	if err != nil {
		return err
	}

	reqBody := map[string]string{
		"conversation_id": conversationID,
	}
	jsonBody, _ := json.Marshal(reqBody)

	url := fmt.Sprintf("%s/api/v1/typing", cfg.ChatServiceUrl)
	log.Printf("[Proxy] Sending typing request to: %s", url)
	
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[Proxy] HTTP error sending typing: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		log.Printf("[Proxy] Chat service returned error status: %d", resp.StatusCode)
		return fmt.Errorf("chat service returned status: %d", resp.StatusCode)
	}

	log.Printf("[Proxy] Typing request successful")
	return nil
}

func generateUserToken(userID, role, secret string) (string, error) {
	claims := jwt.MapClaims{
		"sub":  userID,
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(1 * time.Minute).Unix(), // Short expiry
		"role": role,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
