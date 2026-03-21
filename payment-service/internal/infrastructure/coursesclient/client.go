package coursesclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
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

type CourseInfo struct {
	ID        string  `json:"id"`
	Title     string  `json:"title"`
	Price     float64 `json:"price"`
	Currency  string  `json:"currency"`
	IsPaid    bool    `json:"isPaid"`
	TeacherID string  `json:"teacherId"`
}

func (c *Client) GetCourseByID(ctx context.Context, courseID string) (*CourseInfo, error) {
	url := fmt.Sprintf("%s/api/v1/internal/courses/%s", c.baseURL, courseID)

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	req.Header.Set("x-internal-service-secret", c.internalSecret)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch course: %d", resp.StatusCode)
	}

	var result struct {
		Success bool       `json:"success"`
		Data    CourseInfo `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}

func (c *Client) ActivateEnrollment(ctx context.Context, userID, courseID string) error {
	url := fmt.Sprintf("%s/api/v1/internal/enrollments/activate", c.baseURL)

	reqBody, _ := json.Marshal(map[string]string{
		"userId":   userID,
		"courseId": courseID,
	})

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-internal-service-secret", c.internalSecret)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to activate enrollment: %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) CheckEnrollment(ctx context.Context, userID, courseID string) (bool, error) {
	url := fmt.Sprintf("%s/api/v1/internal/enrollments/check?userId=%s&courseId=%s", c.baseURL, userID, courseID)

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	req.Header.Set("x-internal-service-secret", c.internalSecret)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("failed to check enrollment: %d", resp.StatusCode)
	}

	var result struct {
		Success bool `json:"success"`
		Data    struct {
			IsEnrolled bool `json:"isEnrolled"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}

	return result.Data.IsEnrolled, nil
}

