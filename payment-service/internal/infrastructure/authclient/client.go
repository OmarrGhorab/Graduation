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
)

type Client struct {
	baseURL        string
	internalSecret string
	httpClient     *http.Client
}

func NewClient(baseURL, internalSecret string) *Client {
	return &Client{
		baseURL:        baseURL,
		internalSecret: internalSecret,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type ValidateTokenResponse struct {
	Valid  bool   `json:"valid"`
	UserID string `json:"userId"`
	Role   string `json:"role"`
}

func (c *Client) ValidateToken(ctx context.Context, token string) (*ValidateTokenResponse, error) {
	url := fmt.Sprintf("%s/api/v1/internal/validate-token", c.baseURL)

	reqBody := map[string]string{"token": token}
	body, _ := json.Marshal(reqBody)

	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
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

type UserInfo struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (c *Client) GetUserInfo(ctx context.Context, userID string) (*UserInfo, error) {
	url := fmt.Sprintf("%s/api/v1/internal/users/%s", c.baseURL, userID)

	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
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
		Success bool     `json:"success"`
		Data    UserInfo `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, ErrInvalidResponse
	}

	return &result.Data, nil
}
