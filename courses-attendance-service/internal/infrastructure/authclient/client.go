package authclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

var (
	ErrAuthServiceUnavailable = errors.New("auth service unavailable")
	ErrInvalidResponse        = errors.New("invalid response from auth service")
	ErrUnauthorized           = errors.New("unauthorized")
	ErrVerificationFailed     = errors.New("verification failed")
)

// Client handles communication with the auth service
type Client struct {
	baseURL        string
	internalSecret string
	httpClient     *http.Client
}

// NewClient creates a new auth client
func NewClient(baseURL, internalSecret string) *Client {
	return &Client{
		baseURL:        baseURL,
		internalSecret: internalSecret,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// VerifyContextRequest represents the request to verify attendance context
type VerifyContextRequest struct {
	AccessToken       string `json:"accessToken"`
	DeviceID          string `json:"deviceId"`
	DeviceFingerprint string `json:"deviceFingerprint"`
	AttestationToken  string `json:"attestationToken"`
	IP                string `json:"ip"`
	UserAgent         string `json:"userAgent"`
}

// VerifyContextResponse represents the response from verify-context
type VerifyContextResponse struct {
	Valid                 bool     `json:"valid"`
	UserID                string   `json:"userId"`
	Role                  string   `json:"role"`
	SessionJTI            string   `json:"sessionJti"`
	DeviceVerified        bool     `json:"deviceVerified"`
	EmulatorDetected      bool     `json:"emulatorDetected"`
	MultiDeviceViolation  bool     `json:"multiDeviceViolation"`
	SharedDeviceViolation bool     `json:"sharedDeviceViolation"`
	Reasons               []string `json:"reasons"`
}

// VerifyAttendanceContext verifies the full context for attendance scanning
func (c *Client) VerifyAttendanceContext(ctx context.Context, req VerifyContextRequest) (*VerifyContextResponse, error) {
	url := fmt.Sprintf("%s/api/v1/internal/attendance/verify-context", c.baseURL)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-internal-service-secret", c.internalSecret)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, ErrAuthServiceUnavailable
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, ErrUnauthorized
	}

	if resp.StatusCode != http.StatusOK {
		return nil, ErrInvalidResponse
	}

	var result struct {
		Success bool                  `json:"success"`
		Data    VerifyContextResponse `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, ErrInvalidResponse
	}

	return &result.Data, nil
}

// VerifyParentLinkRequest represents the request to verify parent-child link
type VerifyParentLinkRequest struct {
	ParentID string `json:"parentId"`
	ChildID  string `json:"childId"`
}

// VerifyParentLinkResponse represents the response from verify-link
type VerifyParentLinkResponse struct {
	Valid    bool   `json:"valid"`
	ParentID string `json:"parentId"`
	ChildID  string `json:"childId"`
	Relation string `json:"relation"`
}

// VerifyParentLink verifies if a parent is linked to a child
func (c *Client) VerifyParentLink(ctx context.Context, parentID, childID string) (*VerifyParentLinkResponse, error) {
	url := fmt.Sprintf("%s/api/v1/parent-link/verify-link?parentId=%s&childId=%s", c.baseURL, parentID, childID)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("x-internal-service-secret", c.internalSecret)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, ErrAuthServiceUnavailable
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ErrInvalidResponse
	}

	var result struct {
		Success bool                     `json:"success"`
		Data    VerifyParentLinkResponse `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, ErrInvalidResponse
	}

	return &result.Data, nil
}

// ValidateTokenRequest represents the request to validate a token
type ValidateTokenRequest struct {
	Token string `json:"token"`
}

// ValidateTokenResponse represents the response from token validation
type ValidateTokenResponse struct {
	Valid  bool   `json:"valid"`
	UserID string `json:"userId"`
	Role   string `json:"role"`
}

// ValidateToken validates an access token with the auth service
func (c *Client) ValidateToken(ctx context.Context, token string) (*ValidateTokenResponse, error) {
	url := fmt.Sprintf("%s/api/v1/internal/validate-token", c.baseURL)

	reqBody := ValidateTokenRequest{Token: token}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-internal-service-secret", c.internalSecret)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, ErrAuthServiceUnavailable
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ErrInvalidResponse
	}

	var result ValidateTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, ErrInvalidResponse
	}

	return &result, nil
}

// UserInfo represents user information from auth service
type UserInfo struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	Role       string `json:"role"`
	ProfileImg string `json:"profileImg"`
	Username   string `json:"username"`
}

// ChildInfo represents information about a child linked to a parent
type ChildInfo struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	ProfileImg string `json:"profileImg"`
	Email      string `json:"email"`
	Relation   string `json:"relation"`
}

// GetUserInfo fetches user information by user ID
func (c *Client) GetUserInfo(ctx context.Context, userID string) (*UserInfo, error) {
	url := fmt.Sprintf("%s/api/v1/internal/users/%s", c.baseURL, userID)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("x-internal-service-secret", c.internalSecret)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, ErrAuthServiceUnavailable
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New("user not found")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, ErrInvalidResponse
	}

	var result struct {
		Success bool     `json:"success"`
		Data    UserInfo `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, ErrInvalidResponse
	}

	return &result.Data, nil
}

// GetChildren fetches all children linked to a parent
func (c *Client) GetChildren(ctx context.Context, parentID string) ([]ChildInfo, error) {
	url := fmt.Sprintf("%s/api/v1/parent-link/children?parentId=%s", c.baseURL, parentID)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("x-internal-service-secret", c.internalSecret)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, ErrAuthServiceUnavailable
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ErrInvalidResponse
	}

	var result struct {
		Success bool        `json:"success"`
		Data    []ChildInfo `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, ErrInvalidResponse
	}

	return result.Data, nil
}
