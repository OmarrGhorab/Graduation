package websocket

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gofiber/contrib/websocket"
)

type Client struct {
	Manager *Manager
	UserID  string
	Conn    *websocket.Conn
	Send    chan []byte
}

func (c *Client) ReadPump() {
	defer func() {
		c.Manager.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, msg, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}
		// We can handle upstream messages here (e.g. typing) if we want to proxy them via Gateway
		// For now, simpler to reuse the HTTP API for sending.
		// But let's log it.
		log.Printf("Received raw WS message from %s: %s", c.UserID, string(msg))

		// Optional: Proxy "typing" if we want purely socket-based typing
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(50 * time.Second)
	heartbeat := time.NewTicker(2 * time.Minute) // Refresh presence every 2 minutes
	defer func() {
		ticker.Stop()
		heartbeat.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		case <-heartbeat.C:
			// Refresh user's online status in Redis
			c.Manager.KeepAlive(c.UserID)
		}
	}
}

// Helper to wrap payload in standardized structure
func CreateWSPayload(eventType string, data interface{}) []byte {
	wrapper := struct {
		Type    string      `json:"type"`
		Payload interface{} `json:"payload"`
	}{
		Type:    eventType,
		Payload: data,
	}
	bytes, _ := json.Marshal(wrapper)
	return bytes
}
