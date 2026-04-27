package aiclient

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	serviceURL     string
	internalSecret string
	httpClient     *http.Client
}

func NewClient(serviceURL, internalSecret string) *Client {
	return &Client{
		serviceURL:     serviceURL,
		internalSecret: internalSecret,
		httpClient:     &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *Client) InvalidateRecommendationCache(ctx context.Context, userID string) error {
	url := fmt.Sprintf("%s/api/v1/recommendations/cache/%s", c.serviceURL, userID)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("x-internal-service-secret", c.internalSecret)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("AI service returned status: %d", resp.StatusCode)
	}

	return nil
}
