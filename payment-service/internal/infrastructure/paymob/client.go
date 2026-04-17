package paymob

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type FlexibleBool bool

func (fb *FlexibleBool) UnmarshalJSON(data []byte) error {
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	switch val := v.(type) {
	case bool:
		*fb = FlexibleBool(val)
	case string:
		str := strings.ToLower(strings.TrimSpace(val))
		*fb = FlexibleBool(str == "true" || str == "1" || str == "yes")
	case float64:
		*fb = FlexibleBool(val != 0)
	default:
		*fb = false
	}
	return nil
}

type Client struct {
	apiKey              string
	cardIntegrationID   string
	walletIntegrationID string
	iframeID            string
	hmacSecret          string
	httpClient          *http.Client
	baseURL             string
	redirectURL         string
}

func NewClient(apiKey, cardID, walletID, iframeID, hmacSecret, redirectURL string) *Client {
	return &Client{
		apiKey:              apiKey,
		cardIntegrationID:   cardID,
		walletIntegrationID: walletID,
		iframeID:            iframeID,
		hmacSecret:          hmacSecret,
		redirectURL:         redirectURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "https://accept.paymob.com/api",
	}
}

// 1. Authenticate
func (c *Client) Authenticate(ctx context.Context) (string, error) {
	url := fmt.Sprintf("%s/auth/tokens", c.baseURL)
	reqBody, _ := json.Marshal(map[string]string{"api_key": c.apiKey})

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("auth failed with status %d", resp.StatusCode)
	}

	var result struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Token, nil
}

// 2. Create Order
func (c *Client) CreateOrder(ctx context.Context, authToken string, amountCents int64, currency string) (int64, error) {
	url := fmt.Sprintf("%s/ecommerce/orders", c.baseURL)
	reqBody, _ := json.Marshal(map[string]interface{}{
		"auth_token":      authToken,
		"delivery_needed": "false",
		"amount_cents":    amountCents,
		"currency":        currency,
		"items":           []interface{}{},
	})

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("create order failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		ID int64 `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	return result.ID, nil
}

// 3. Create Payment Key
type BillingData struct {
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Email       string `json:"email"`
	PhoneNumber string `json:"phone_number"`
	Apartment   string `json:"apartment"`
	Floor       string `json:"floor"`
	Street      string `json:"street"`
	Building    string `json:"building"`
	City        string `json:"city"`
	Country     string `json:"country"`
	State       string `json:"state"`
}

func (c *Client) CreatePaymentKey(ctx context.Context, authToken string, orderID int64, amountCents int64, currency string, integrationID string, billing BillingData, tokenization bool) (string, error) {
	url := fmt.Sprintf("%s/acceptance/payment_keys", c.baseURL)
	reqBody, _ := json.Marshal(map[string]interface{}{
		"auth_token":           authToken,
		"amount_cents":         amountCents,
		"expiration":           3600,
		"order_id":             fmt.Sprintf("%d", orderID),
		"billing_data":         billing,
		"currency":             currency,
		"integration_id":       integrationID,
		"lock_order_when_paid": "false",
		"tokenization":         tokenization,
		"redirection_url":      c.redirectURL,
	})

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("create payment key failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Token, nil
}

type PayWithTokenResponse struct {
	ID                int64        `json:"id"`
	Success           FlexibleBool `json:"success"`
	Pending           FlexibleBool `json:"pending"`
	RedirectURL       string       `json:"redirection_url"`
	IframeRedirectURL string       `json:"iframe_redirection_url"`
}

func (c *Client) PayWithToken(ctx context.Context, paymentToken string, cardToken string) (*PayWithTokenResponse, error) {
	url := fmt.Sprintf("%s/acceptance/payments/pay", c.baseURL)
	reqBody, _ := json.Marshal(map[string]interface{}{
		"source": map[string]string{
			"identifier": cardToken,
			"subtype":    "TOKEN",
		},
		"payment_token": paymentToken,
	})

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("[Paymob Debug] Token Pay Response: %s\n", string(body))

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("token pay failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result PayWithTokenResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// 4. Wallet Pay
func (c *Client) PayWithWallet(ctx context.Context, paymentToken string, phoneNumber string) (string, error) {
	url := fmt.Sprintf("%s/acceptance/payments/pay", c.baseURL)
	reqBody, _ := json.Marshal(map[string]interface{}{
		"source": map[string]string{
			"identifier": phoneNumber,
			"subtype":    "WALLET",
		},
		"payment_token": paymentToken,
	})

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("wallet pay failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		RedirectURL       string `json:"redirection_url"`
		IframeRedirectURL string `json:"iframe_redirection_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.RedirectURL != "" {
		return result.RedirectURL, nil
	}
	return result.IframeRedirectURL, nil

}

// Helper: Build Iframe URL
func (c *Client) GetCardPaymentURL(paymentToken string) string {
	return fmt.Sprintf("https://accept.paymob.com/api/acceptance/iframes/%s?payment_token=%s", c.iframeID, paymentToken)
}

// HMAC Verification
func (c *Client) VerifyHMAC(hmacHeader string, data map[string]interface{}) (bool, error) {
	// Paymob HMAC verification logic
	// If this is a webhook, actual data is in "obj"
	sourceData := data
	if obj, ok := data["obj"].(map[string]interface{}); ok {
		sourceData = obj
	}

	// Keys to include in HMAC calculation (order matters)
	var keys []string
	eventType, _ := data["type"].(string)

	if eventType == "TOKEN" {
		keys = []string{
			"card_subtype",
			"created_at",
			"email",
			"id",
			"masked_pan",
			"merchant_id",
			"order_id",
			"token",
		}
	} else {
		keys = []string{
			"amount_cents",
			"created_at",
			"currency",
			"error_occured",
			"has_parent_transaction",
			"id",
			"integration_id",
			"is_3d_secure",
			"is_auth",
			"is_capture",
			"is_refunded",
			"is_standalone_payment",
			"is_voided",
			"order",
			"owner",
			"pending",
			"source_data.pan",
			"source_data.sub_type",
			"source_data.type",
			"success",
		}
	}

	var values []string
	for _, key := range keys {
		val := getNestedValue(sourceData, key)

		// Fix for "order" being a map in webhooks
		if key == "order" {
			if m, ok := val.(map[string]interface{}); ok {
				val = m["id"]
			}
		}

		var valStr string
		switch v := val.(type) {
		case nil:
			valStr = ""
		case bool:
			valStr = fmt.Sprintf("%v", v) // "true" or "false"
		case float64:
			// JSON numbers are float64. Format as integer to avoid scientific notation.
			valStr = fmt.Sprintf("%.0f", v)
		case int, int64:
			valStr = fmt.Sprintf("%d", v)
		default:
			valStr = fmt.Sprintf("%v", v)
		}
		values = append(values, valStr)
	}

	concatString := strings.Join(values, "")
	// fmt.Printf("[Paymob Debug] HMAC Concat String: %s\n", concatString)

	h := hmac.New(sha512.New, []byte(c.hmacSecret))
	h.Write([]byte(concatString))
	expectedHMAC := hex.EncodeToString(h.Sum(nil))

	isValid := strings.EqualFold(hmacHeader, expectedHMAC)
	if !isValid {
		// Log detailed info for debugging if needed
		fmt.Printf("[Paymob Debug] HMAC Mismatch!\n")
		fmt.Printf("  Concat: %s\n", concatString)
		fmt.Printf("  Received: %s\n", hmacHeader)
		fmt.Printf("  Expected: %s\n", expectedHMAC)
	}

	return isValid, nil
}

func getNestedValue(data map[string]interface{}, path string) interface{} {
	parts := strings.Split(path, ".")
	var current interface{} = data
	for _, part := range parts {
		if m, ok := current.(map[string]interface{}); ok {
			current = m[part]
		} else {
			return nil
		}
	}
	return current
}

func (c *Client) GetCardIntegrationID() string   { return c.cardIntegrationID }
func (c *Client) GetWalletIntegrationID() string { return c.walletIntegrationID }
