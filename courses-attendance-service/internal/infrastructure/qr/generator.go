package qr

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/clock"
	"github.com/google/uuid"
)

var (
	ErrInvalidSignature = errors.New("invalid QR signature")
	ErrTokenExpired     = errors.New("QR token has expired")
	ErrInvalidPayload   = errors.New("invalid QR payload")
)

// Generator handles QR code token generation and validation
type Generator struct {
	signingSecret string
	clock         clock.Clock
}

// NewGenerator creates a new QR generator
func NewGenerator(signingSecret string, clk clock.Clock) *Generator {
	return &Generator{
		signingSecret: signingSecret,
		clock:         clk,
	}
}

// TokenPayload represents the data encoded in a QR code
type TokenPayload struct {
	LessonID  string    `json:"lid"`
	Nonce     string    `json:"n"`
	IssuedAt  time.Time `json:"iat"`
	ExpiresAt time.Time `json:"exp"`
}

// SignedToken represents a complete signed QR token
type SignedToken struct {
	Payload   TokenPayload `json:"payload"`
	Signature string       `json:"sig"`
	Raw       string       `json:"raw"` // Base64 encoded payload for QR display
}

// GenerateToken creates a new signed QR token for a lesson
func (g *Generator) GenerateToken(lessonID uuid.UUID, rotationSeconds, expirySeconds int) (*SignedToken, error) {
	now := g.clock.Now()

	// Generate cryptographically secure nonce
	nonce, err := generateNonce(16)
	if err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	payload := TokenPayload{
		LessonID:  lessonID.String(),
		Nonce:     nonce,
		IssuedAt:  now,
		ExpiresAt: now.Add(time.Duration(expirySeconds) * time.Second),
	}

	// Serialize payload
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create canonical string for signing
	canonical := g.createCanonicalString(payload)

	// Sign the canonical string
	signature := g.sign(canonical)

	// Encode payload for QR display
	raw := base64.URLEncoding.EncodeToString(payloadBytes)

	return &SignedToken{
		Payload:   payload,
		Signature: signature,
		Raw:       raw,
	}, nil
}

// ValidateToken validates a QR token
func (g *Generator) ValidateToken(raw, signature string, serverTime time.Time) (*TokenPayload, error) {
	// Decode payload
	payloadBytes, err := base64.URLEncoding.DecodeString(raw)
	if err != nil {
		return nil, ErrInvalidPayload
	}

	var payload TokenPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, ErrInvalidPayload
	}

	// Validate signature
	canonical := g.createCanonicalString(payload)
	expectedSignature := g.sign(canonical)

	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return nil, ErrInvalidSignature
	}

	// Check expiry using server time (never trust client time)
	if serverTime.After(payload.ExpiresAt) {
		return nil, ErrTokenExpired
	}

	return &payload, nil
}

// createCanonicalString creates a canonical string for signing
func (g *Generator) createCanonicalString(payload TokenPayload) string {
	return fmt.Sprintf("%s|%s|%d|%d",
		payload.LessonID,
		payload.Nonce,
		payload.IssuedAt.Unix(),
		payload.ExpiresAt.Unix(),
	)
}

// sign creates an HMAC-SHA256 signature
func (g *Generator) sign(data string) string {
	h := hmac.New(sha256.New, []byte(g.signingSecret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// generateNonce generates a cryptographically secure random nonce
func generateNonce(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// QRCodeData represents the data to be encoded in the QR code
type QRCodeData struct {
	Raw       string `json:"r"`
	Signature string `json:"s"`
}

// ToQRString converts a signed token to a string suitable for QR encoding
func (t *SignedToken) ToQRString() (string, error) {
	data := QRCodeData{
		Raw:       t.Raw,
		Signature: t.Signature,
	}
	bytes, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// ParseQRString parses a QR code string
func ParseQRString(qrString string) (*QRCodeData, error) {
	var data QRCodeData
	if err := json.Unmarshal([]byte(qrString), &data); err != nil {
		return nil, ErrInvalidPayload
	}
	return &data, nil
}
