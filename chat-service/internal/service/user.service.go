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

type GroupInfo struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Image string `json:"image"`
}

type ChatContexts struct {
	Teachers   []string    `json:"teachers"`
	Students   []string    `json:"students"`
	Assistants []string    `json:"assistants"`
	Groups     []GroupInfo `json:"groups"`
}

type UserService struct {
	authServiceURL        string
	coursesServiceURL     string
	internalServiceSecret string
	httpClient            *http.Client
}

func NewUserService(authServiceURL, coursesServiceURL, internalServiceSecret string) *UserService {
	return &UserService{
		authServiceURL:        authServiceURL,
		coursesServiceURL:     coursesServiceURL,
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

// FetchParents fetches parent profiles for a student
func (s *UserService) FetchParents(studentID string) ([]*UserProfile, error) {
	url := fmt.Sprintf("%s/api/v1/internal/users/%s/parents", s.authServiceURL, studentID)
	return s.fetchProfilesList(url)
}

// FetchChildren fetches child profiles for a parent
func (s *UserService) FetchChildren(parentID string) ([]*UserProfile, error) {
	url := fmt.Sprintf("%s/api/v1/internal/users/%s/children", s.authServiceURL, parentID)
	return s.fetchProfilesList(url)
}

// FetchChatContexts fetches academic relationships from courses service
func (s *UserService) FetchChatContexts(userID, role string) (*ChatContexts, error) {
	url := fmt.Sprintf("%s/api/v1/internal/users/%s/chat-contexts?role=%s", s.coursesServiceURL, userID, role)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("x-internal-service-secret", s.internalServiceSecret)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("courses service returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Success bool          `json:"success"`
		Data    *ChatContexts `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Data, nil
}

func (s *UserService) fetchProfilesList(url string) ([]*UserProfile, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("x-internal-service-secret", s.internalServiceSecret)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("service returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Success bool           `json:"success"`
		Data    []*UserProfile `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Data, nil
}

