package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

// Config - matching your .env
const (
	ChatServiceURL = "http://localhost:6004/api/v1"
	WSGatewayURL   = "ws://localhost:8001/ws"
	JWTSecret      = "47079cb745e7b5c1ebaa8106cabc00f43d6356ce3b0d4b725156742f4084280b"
)

func main() {
	log.Println("Starting End-to-End Chat Test...")

	// 1. Setup Users
	userA := "11111111-1111-1111-1111-111111111111"
	userB := "22222222-2222-2222-2222-222222222222"

	tokenA := generateToken(userA)
	tokenB := generateToken(userB)

	// 2. Connect User B to WebSocket (Receiver)
	log.Println("[Step 1] Connecting User B to WebSocket...")
	wsB, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("%s?token=%s", WSGatewayURL, tokenB), nil)
	if err != nil {
		log.Fatalf("Failed to connect User B to WS: %v", err)
	}
	defer wsB.Close()

	// Start listening for messages for User B
	done := make(chan bool)
	go func() {
		for {
			_, message, err := wsB.ReadMessage()
			if err != nil {
				log.Printf("User B WS Error: %v", err)
				return
			}
			log.Printf("[SUCCESS] User B Received: %s", string(message))
			done <- true
		}
	}()

	// 3. User A creates a Direct Conversation with User B
	log.Println("[Step 2] User A creating conversation with User B...")
	convID, err := createConversation(tokenA, userB)
	if err != nil {
		log.Fatalf("Failed to create conversation: %v", err)
	}
	log.Printf("Conversation Created: %s", convID)

	// 4. User A sends a message
	log.Println("[Step 3] User A sending message...")
	if err := sendMessage(tokenA, convID, "Hello Automated World!"); err != nil {
		log.Fatalf("Failed to send message: %v", err)
	}

	// 5. Wait for User B to receive
	log.Println("[Step 4] Waiting for receipt...")
	select {
	case <-done:
		log.Println("Test Passed! \u2705")
	case <-time.After(10 * time.Second):
		log.Fatalf("Test Failed: Timed out waiting for message \u274C")
	}
}

// Helpers

func generateToken(userID string) string {
	claims := jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, _ := token.SignedString([]byte(JWTSecret))
	return s
}

func createConversation(token, peerID string) (string, error) {
	body := map[string]string{"peer_id": peerID}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", ChatServiceURL+"/conversations/direct", bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}

	var res struct {
		ID string `json:"id"`
	}
	json.NewDecoder(resp.Body).Decode(&res)
	return res.ID, nil
}

func sendMessage(token, convID, content string) error {
	body := map[string]string{"content": content, "type": "text"}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/conversations/%s/messages", ChatServiceURL, convID), bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return nil
}
