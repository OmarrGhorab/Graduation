package services

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
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
	Folder      string `json:"folder"`
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
	MaxVoiceSize = 15 * 1024 * 1024 // 15 MB
	MaxBatchSize = 20 * 1024 * 1024 // 20 MB (Total for multiple files)
)

// GeneratePresignedURL generates a presigned URL for Cloudinary upload
func (s *MediaService) GeneratePresignedURL(mediaType MediaType, contentType string, fileSize int64) (*PresignResponse, error) {
	// Validate file size
	if err := s.ValidateMediaSize(mediaType, fileSize); err != nil {
		return nil, err
	}

	return s.createPresignResponse(mediaType)
}

// GenerateBatchPresignedURLs generates multiple presigned URLs and validates total size
func (s *MediaService) GenerateBatchPresignedURLs(requests []struct {
	Type        MediaType
	ContentType string
	FileSize    int64
}) ([]*PresignResponse, error) {
	var totalSize int64
	var responses []*PresignResponse

	// First pass: validate sizes
	for _, req := range requests {
		if err := s.ValidateMediaSize(req.Type, req.FileSize); err != nil {
			return nil, err
		}
		totalSize += req.FileSize
	}

	// Validate total batch size
	if totalSize > MaxBatchSize {
		return nil, fmt.Errorf("total batch size %s exceeds maximum of 20MB", formatBytes(totalSize))
	}

	// Second pass: generate URLs
	for _, req := range requests {
		resp, err := s.createPresignResponse(req.Type)
		if err != nil {
			return nil, err
		}
		responses = append(responses, resp)
	}

	return responses, nil
}

func (s *MediaService) createPresignResponse(mediaType MediaType) (*PresignResponse, error) {
	// Generate unique public ID without folder (folder will be passed separately)
	timestamp := time.Now().Unix()
	folder := "chat/" + time.Now().Format("2006/01")
	fileName := fmt.Sprintf("%d_%s", timestamp, uuid.New().String()[:8])

	// Generate signature for Cloudinary
	// Parameters must be sorted alphabetically: folder, public_id, timestamp
	signatureStr := fmt.Sprintf("folder=%s&public_id=%s&timestamp=%d%s", folder, fileName, timestamp, s.apiSecret)
	hash := sha1.Sum([]byte(signatureStr))
	signature := hex.EncodeToString(hash[:])

	// Full public ID for download URL
	fullPublicID := folder + "/" + fileName

	// Determine resource type
	resourceType := "image"
	if mediaType == MediaTypeVoice {
		resourceType = "video" // Cloudinary uses 'video' for audio
	}

	uploadURL := fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/%s/upload", s.cloudName, resourceType)
	downloadURL := fmt.Sprintf("https://res.cloudinary.com/%s/%s/upload/%s", s.cloudName, resourceType, fullPublicID)

	return &PresignResponse{
		UploadURL:   uploadURL,
		Signature:   signature,
		Timestamp:   timestamp,
		APIKey:      s.apiKey,
		Folder:      folder,
		PublicID:    fileName,
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
			return fmt.Errorf("voice size %s exceeds maximum of 15MB", formatBytes(fileSize))
		}
	default:
		return fmt.Errorf("unsupported media type: %s", mediaType)
	}
	return nil
}

// DeleteMedia deletes an asset from Cloudinary given its download URL
func (s *MediaService) DeleteMedia(ctx context.Context, mediaURL string) error {
	if mediaURL == "" {
		return nil
	}

	// Extract resource type and public ID from URL
	// Format: https://res.cloudinary.com/<cloud_name>/<resource_type>/upload/<folder>/<public_id>
	u, err := url.Parse(mediaURL)
	if err != nil {
		return err
	}

	parts := strings.Split(u.Path, "/")
	if len(parts) < 5 {
		return fmt.Errorf("invalid cloudinary URL format: %s", mediaURL)
	}

	// parts[0] is empty
	// parts[1] is cloud_name
	// parts[2] is resource_type (image/video)
	// parts[3] is "upload"
	// parts[4:] is the public_id (including folders)
	resourceType := parts[2]
	publicIDWithFolders := strings.Join(parts[4:], "/")

	// Remove file extension if present
	if idx := strings.LastIndex(publicIDWithFolders, "."); idx != -1 {
		publicIDWithFolders = publicIDWithFolders[:idx]
	}

	timestamp := time.Now().Unix()
	signatureStr := fmt.Sprintf("public_id=%s&timestamp=%d%s", publicIDWithFolders, timestamp, s.apiSecret)
	hash := sha1.Sum([]byte(signatureStr))
	signature := hex.EncodeToString(hash[:])

	endpoint := fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/%s/destroy", s.cloudName, resourceType)

	data := url.Values{}
	data.Set("public_id", publicIDWithFolders)
	data.Set("timestamp", strconv.FormatInt(timestamp, 10))
	data.Set("api_key", s.apiKey)
	data.Set("signature", signature)

	resp, err := http.PostForm(endpoint, data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("cloudinary destroy failed with status: %d", resp.StatusCode)
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
