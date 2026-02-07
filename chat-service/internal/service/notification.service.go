package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type NotificationService struct {
	baseURL        string
	internalSecret string
	httpClient     *http.Client
}

func NewNotificationService(baseURL, internalSecret string) *NotificationService {
	return &NotificationService{
		baseURL:        baseURL,
		internalSecret: internalSecret,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type NotificationPayload struct {
	UserID string                 `json:"userId"`
	Type   string                 `json:"type"`
	Data   map[string]interface{} `json:"data"`
}

func (ns *NotificationService) SendNotification(payload NotificationPayload) error {
	url := fmt.Sprintf("%s/api/v1/notifications/publish", ns.baseURL)
	
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-internal-service-secret", ns.internalSecret)

	resp, err := ns.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("notification service returned status %d", resp.StatusCode)
	}

	fmt.Printf("[NotificationService] Sent notification to user %s, type: %s\n", payload.UserID, payload.Type)
	return nil
}
