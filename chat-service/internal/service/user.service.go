package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type UserProfile struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Image string `json:"image"`
	Role  string `json:"role"`
}

type UserService struct {
	authServiceURL        string
	internalServiceSecret string
	httpClient            *http.Client
}

func NewUserService(authServiceURL, internalServiceSecret string) *UserService {
	return &UserService{
		authServiceURL:        authServiceURL,
		internalServiceSecret: internalServiceSecret,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// FetchUserProfiles fetches multiple user profiles from auth service using batch endpoint
func (s *UserService) FetchUserProfiles(userIDs []string) (map[string]*UserProfile, error) {
	if len(userIDs) == 0 {
		return make(map[string]*UserProfile), nil
	}

	url := fmt.Sprintf("%s/api/v1/internal/users/batch", s.authServiceURL)

	// Prepare request body
	requestBody := map[string]interface{}{
		"userIds": userIDs,
	}
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-internal-service-secret", s.internalServiceSecret)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("auth service returned status %d: %s", resp.StatusCode, string(body))
	}

	var profiles map[string]*UserProfile
	if err := json.NewDecoder(resp.Body).Decode(&profiles); err != nil {
		return nil, err
	}

	// Add fallback for missing users
	for _, userID := range userIDs {
		if profiles[userID] == nil {
			profiles[userID] = &UserProfile{
				ID:    userID,
				Name:  "Unknown User",
				Image: "",
				Role:  "STUDENT",
			}
		}
	}

	return profiles, nil
}

