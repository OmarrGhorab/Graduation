package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"time"
)

type EmailService struct {
	resendAPIKey string
	fromEmail    string
	fromName     string
	httpClient   *http.Client
}

func NewEmailService(resendAPIKey, fromEmail, fromName string) *EmailService {
	return &EmailService{
		resendAPIKey: resendAPIKey,
		fromEmail:    fromEmail,
		fromName:     fromName,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type SubscriptionRenewalEmail struct {
	UserName        string
	CourseName      string
	Amount          string
	Currency        string
	NextBillingDate string
	PaymentURL      string
	SubscriptionID  string
}

type PaymentReceiptEmail struct {
	UserName    string
	OrderID     string
	Courses     []string
	Amount      string
	Currency    string
	PaymentDate string
}

// Resend API request/response structures
type resendEmailRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
}

type resendEmailResponse struct {
	ID string `json:"id"`
}

type resendErrorResponse struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
	Name       string `json:"name"`
}

func (s *EmailService) SendSubscriptionRenewalNotification(ctx context.Context, to string, data SubscriptionRenewalEmail) error {
	subject := "Your Subscription Renewal is Due"
	
	htmlTemplate := `
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #4CAF50; color: white; padding: 20px; text-align: center; border-radius: 8px 8px 0 0; }
        .content { padding: 20px; background-color: #f9f9f9; border: 1px solid #ddd; border-top: none; }
        .button { display: inline-block; padding: 12px 24px; background-color: #4CAF50; color: white; text-decoration: none; border-radius: 4px; margin: 20px 0; }
        .footer { text-align: center; padding: 20px; font-size: 12px; color: #666; }
        .amount { font-size: 24px; font-weight: bold; color: #4CAF50; }
        .info-box { background: white; padding: 15px; border-radius: 8px; margin: 15px 0; border-left: 4px solid #4CAF50; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>💳 Subscription Renewal Due</h1>
        </div>
        <div class="content">
            <p>Hi {{.UserName}},</p>
            <p>Your monthly subscription for <strong>{{.CourseName}}</strong> is due for renewal.</p>
            
            <div class="info-box">
                <p><strong>Amount:</strong> <span class="amount">{{.Amount}} {{.Currency}}</span></p>
                <p><strong>Next Billing Date:</strong> {{.NextBillingDate}}</p>
                <p><strong>Subscription ID:</strong> {{.SubscriptionID}}</p>
            </div>
            
            <p>Please complete your payment to continue accessing the course:</p>
            <div style="text-align: center;">
                <a href="{{.PaymentURL}}" class="button">Pay Now</a>
            </div>
            
            <p style="margin-top: 20px;">If you wish to cancel your subscription, you can do so from your account settings.</p>
        </div>
        <div class="footer">
            <p>This is an automated message from Payment Service.</p>
            <p>Please do not reply to this email.</p>
        </div>
    </div>
</body>
</html>
`

	tmpl, err := template.New("renewal").Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	var body bytes.Buffer
	if err := tmpl.Execute(&body, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return s.sendEmail(ctx, to, subject, body.String())
}

func (s *EmailService) SendPaymentReceipt(ctx context.Context, to string, data PaymentReceiptEmail) error {
	subject := "Payment Receipt - Order Confirmation"
	
	htmlTemplate := `
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #2196F3; color: white; padding: 20px; text-align: center; border-radius: 8px 8px 0 0; }
        .content { padding: 20px; background-color: #f9f9f9; border: 1px solid #ddd; border-top: none; }
        .success { color: #4CAF50; font-size: 18px; font-weight: bold; text-align: center; margin: 20px 0; }
        .order-details { background-color: white; padding: 15px; margin: 20px 0; border-left: 4px solid #2196F3; border-radius: 4px; }
        .course-list { list-style: none; padding: 0; }
        .course-list li { padding: 8px 0; border-bottom: 1px solid #eee; }
        .amount { font-size: 24px; font-weight: bold; color: #2196F3; }
        .footer { text-align: center; padding: 20px; font-size: 12px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>✓ Payment Successful</h1>
        </div>
        <div class="content">
            <p>Hi {{.UserName}},</p>
            <p class="success">Your payment has been processed successfully!</p>
            
            <div class="order-details">
                <p><strong>Order ID:</strong> {{.OrderID}}</p>
                <p><strong>Payment Date:</strong> {{.PaymentDate}}</p>
                <p><strong>Amount Paid:</strong> <span class="amount">{{.Amount}} {{.Currency}}</span></p>
                
                <p><strong>Courses:</strong></p>
                <ul class="course-list">
                    {{range .Courses}}
                    <li>{{.}}</li>
                    {{end}}
                </ul>
            </div>
            
            <p>You now have full access to your enrolled courses. Happy learning!</p>
        </div>
        <div class="footer">
            <p>Thank you for your purchase!</p>
            <p>This is an automated message from Payment Service.</p>
            <p>Please do not reply to this email.</p>
        </div>
    </div>
</body>
</html>
`

	tmpl, err := template.New("receipt").Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	var body bytes.Buffer
	if err := tmpl.Execute(&body, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return s.sendEmail(ctx, to, subject, body.String())
}

func (s *EmailService) SendSubscriptionCancellationConfirmation(ctx context.Context, to, userName, courseName, subscriptionID string) error {
	subject := "Subscription Cancelled"
	
	htmlTemplate := `
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #FF9800; color: white; padding: 20px; text-align: center; border-radius: 8px 8px 0 0; }
        .content { padding: 20px; background-color: #f9f9f9; border: 1px solid #ddd; border-top: none; }
        .footer { text-align: center; padding: 20px; font-size: 12px; color: #666; }
        .info-box { background: white; padding: 15px; border-radius: 8px; margin: 15px 0; border-left: 4px solid #FF9800; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Subscription Cancelled</h1>
        </div>
        <div class="content">
            <p>Hi {{.UserName}},</p>
            <p>Your subscription for <strong>{{.CourseName}}</strong> has been cancelled successfully.</p>
            
            <div class="info-box">
                <p>You will continue to have access until the end of your current billing period.</p>
                <p><strong>Subscription ID:</strong> {{.SubscriptionID}}</p>
            </div>
            
            <p>We're sorry to see you go! If you change your mind, you can always resubscribe.</p>
        </div>
        <div class="footer">
            <p>This is an automated message from Payment Service.</p>
            <p>Please do not reply to this email.</p>
        </div>
    </div>
</body>
</html>
`

	tmpl, err := template.New("cancellation").Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	data := struct {
		UserName       string
		CourseName     string
		SubscriptionID string
	}{
		UserName:       userName,
		CourseName:     courseName,
		SubscriptionID: subscriptionID,
	}

	var body bytes.Buffer
	if err := tmpl.Execute(&body, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return s.sendEmail(ctx, to, subject, body.String())
}

func (s *EmailService) sendEmail(ctx context.Context, to, subject, htmlBody string) error {
	// If Resend API key is not configured, just log the email
	if s.resendAPIKey == "" {
		log.Printf("EMAIL (not sent - Resend API key not configured):\nTo: %s\nSubject: %s\n", to, subject)
		return nil
	}

	// Prepare Resend API request
	reqBody := resendEmailRequest{
		From:    fmt.Sprintf("%s <%s>", s.fromName, s.fromEmail),
		To:      []string{to},
		Subject: subject,
		HTML:    htmlBody,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.resend.com/emails", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.resendAPIKey))
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("Failed to send email to %s: %v", to, err)
		return fmt.Errorf("failed to send email: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Check response status
	if resp.StatusCode != http.StatusOK {
		var errResp resendErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err == nil {
			log.Printf("Resend API error: %s - %s", errResp.Name, errResp.Message)
			return fmt.Errorf("resend API error: %s", errResp.Message)
		}
		log.Printf("Failed to send email to %s: HTTP %d - %s", to, resp.StatusCode, string(respBody))
		return fmt.Errorf("failed to send email: HTTP %d", resp.StatusCode)
	}

	// Parse success response
	var successResp resendEmailResponse
	if err := json.Unmarshal(respBody, &successResp); err != nil {
		log.Printf("Warning: Failed to parse success response: %v", err)
	}

	log.Printf("Email sent successfully to %s (ID: %s)", to, successResp.ID)
	return nil
}
