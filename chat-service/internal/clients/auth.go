package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/graduation/chat-service/internal/models"
)

// AuthClient handles communication with the auth-service
type AuthClient struct {
	serviceURL     string
	internalSecret string
	httpClient     *http.Client
}

// NewAuthClient creates a new AuthClient
func NewAuthClient(serviceURL, internalSecret string) *AuthClient {
	return &AuthClient{
		serviceURL:     serviceURL,
		internalSecret: internalSecret,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// UserProfile represents the user data returned by auth-service
type UserProfile struct {
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Image string          `json:"image"`
	Role  models.UserRole `json:"role"`
}

// GetBatchUsers fetches details for multiple users
func (c *AuthClient) GetBatchUsers(ctx context.Context, userIDs []string) (map[string]UserProfile, error) {
	if len(userIDs) == 0 {
		return make(map[string]UserProfile), nil
	}

	payload := map[string]interface{}{
		"userIds": userIDs,
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.serviceURL+"/api/v1/internal/users/batch", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-internal-service-secret", c.internalSecret)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		fmt.Printf("[AuthClient] Request failed: %v\n", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("[AuthClient] Status: %d\n", resp.StatusCode)
		return nil, fmt.Errorf("auth service returned status: %d", resp.StatusCode)
	}

	// Read body for debugging
	var users map[string]UserProfile
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		fmt.Printf("[AuthClient] Decode error: %v\n", err)
		return nil, err
	}

	// fmt.Printf("[AuthClient] Received users: %+v\n", users) // Optional: Uncomment to see full data
	return users, nil
}
