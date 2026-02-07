package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type PresenceService struct {
	wsGatewayURL string
	httpClient   *http.Client
}

func NewPresenceService(wsGatewayURL string) *PresenceService {
	return &PresenceService{
		wsGatewayURL: wsGatewayURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// CheckPresence checks if users are online via ws-gateway
func (ps *PresenceService) CheckPresence(userIDs []string) (map[string]bool, error) {
	if len(userIDs) == 0 {
		return make(map[string]bool), nil
	}

	reqBody := map[string]interface{}{
		"user_ids": userIDs,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/api/presence", ps.wsGatewayURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := ps.httpClient.Do(req)
	if err != nil {
		// If ws-gateway is down, return all offline
		fmt.Printf("Error checking presence: %v\n", err)
		result := make(map[string]bool)
		for _, id := range userIDs {
			result[id] = false
		}
		return result, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ws-gateway returned status %d: %s", resp.StatusCode, string(body))
	}

	var response struct {
		Presence map[string]bool `json:"presence"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return response.Presence, nil
}
