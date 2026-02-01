package services

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"
)

// MediaService handles media upload logic with Cloudinary
type MediaService struct {
	cloudName string
	apiKey    string
	apiSecret string
}

// NewMediaService creates a new MediaService
func NewMediaService(cloudName, apiKey, apiSecret string) *MediaService {
	return &MediaService{
		cloudName: cloudName,
		apiKey:    apiKey,
		apiSecret: apiSecret,
	}
}

// PresignResponse is the response for a presign request
type PresignResponse struct {
	UploadURL   string `json:"upload_url"`
	Signature   string `json:"signature"`
	Timestamp   int64  `json:"timestamp"`
	APIKey      string `json:"api_key"`
	PublicID    string `json:"public_id"`
	DownloadURL string `json:"download_url"`
}

// MediaType for validation
type MediaType string

const (
	MediaTypeImage MediaType = "image"
	MediaTypeVoice MediaType = "voice"
)

// Size limits
const (
	MaxImageSize = 5 * 1024 * 1024  // 5 MB
	MaxVoiceSize = 10 * 1024 * 1024 // 10 MB
)

// GeneratePresignedURL generates a presigned URL for Cloudinary upload
func (s *MediaService) GeneratePresignedURL(mediaType MediaType, contentType string, fileSize int64) (*PresignResponse, error) {
	// Validate file size
	if mediaType == MediaTypeImage && fileSize > MaxImageSize {
		return nil, fmt.Errorf("image size exceeds maximum of 5MB")
	}
	if mediaType == MediaTypeVoice && fileSize > MaxVoiceSize {
		return nil, fmt.Errorf("voice size exceeds maximum of 10MB")
	}

	// Generate unique public ID
	timestamp := time.Now().Unix()
	folder := "chat/" + time.Now().Format("2006/01")
	publicID := fmt.Sprintf("%s/%d", folder, timestamp)

	// Generate signature for Cloudinary
	signatureStr := fmt.Sprintf("public_id=%s&timestamp=%d%s", publicID, timestamp, s.apiSecret)
	hash := sha1.Sum([]byte(signatureStr))
	signature := hex.EncodeToString(hash[:])

	// Determine resource type
	resourceType := "image"
	if mediaType == MediaTypeVoice {
		resourceType = "video" // Cloudinary uses 'video' for audio
	}

	uploadURL := fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/%s/upload", s.cloudName, resourceType)
	downloadURL := fmt.Sprintf("https://res.cloudinary.com/%s/%s/upload/%s", s.cloudName, resourceType, publicID)

	return &PresignResponse{
		UploadURL:   uploadURL,
		Signature:   signature,
		Timestamp:   timestamp,
		APIKey:      s.apiKey,
		PublicID:    publicID,
		DownloadURL: downloadURL,
	}, nil
}

// ValidateMediaSize validates media size based on type
func (s *MediaService) ValidateMediaSize(mediaType MediaType, fileSize int64) error {
	switch mediaType {
	case MediaTypeImage:
		if fileSize > MaxImageSize {
			return fmt.Errorf("image size %s exceeds maximum of 5MB", formatBytes(fileSize))
		}
	case MediaTypeVoice:
		if fileSize > MaxVoiceSize {
			return fmt.Errorf("voice size %s exceeds maximum of 10MB", formatBytes(fileSize))
		}
	default:
		return fmt.Errorf("unsupported media type: %s", mediaType)
	}
	return nil
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return strconv.FormatInt(bytes, 10) + " B"
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
