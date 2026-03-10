package cloudinary

import (
	"context"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/OmarrGhorab/courses-attendance-service/internal/config"
)

// Client wraps Cloudinary SDK
type Client struct {
	cld    *cloudinary.Cloudinary
	folder string
}

// UploadResult contains information about uploaded file
type UploadResult struct {
	URL          string
	PublicID     string
	ResourceType string
	Format       string
	Duration     *int // For videos, duration in seconds
	Bytes        int64
}

// NewClient creates a new Cloudinary client
func NewClient(cfg config.CloudinaryConfig) (*Client, error) {
	cld, err := cloudinary.NewFromParams(cfg.CloudName, cfg.APIKey, cfg.APISecret)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Cloudinary: %w", err)
	}

	return &Client{
		cld:    cld,
		folder: cfg.Folder,
	}, nil
}

// UploadVideo uploads a video file to Cloudinary
func (c *Client) UploadVideo(ctx context.Context, file multipart.File, filename string) (*UploadResult, error) {
	// Generate unique public ID
	publicID := c.generatePublicID(filename, "videos")

	overwrite := false
	uploadParams := uploader.UploadParams{
		PublicID:     publicID,
		Folder:       c.folder,
		ResourceType: "video",
		Overwrite:    &overwrite,
	}

	result, err := c.cld.Upload.Upload(ctx, file, uploadParams)
	if err != nil {
		return nil, fmt.Errorf("failed to upload video: %w", err)
	}

	uploadResult := &UploadResult{
		URL:          result.SecureURL,
		PublicID:     result.PublicID,
		ResourceType: result.ResourceType,
		Format:       result.Format,
		Bytes:        int64(result.Bytes),
	}

	// Note: Duration extraction from Cloudinary response varies by SDK version
	// For now, duration will be nil and can be set manually if needed

	return uploadResult, nil
}

// UploadDocument uploads a document (PDF, DOCX, etc.) to Cloudinary
func (c *Client) UploadDocument(ctx context.Context, file multipart.File, filename string) (*UploadResult, error) {
	// Generate unique public ID
	publicID := c.generatePublicID(filename, "documents")

	overwrite := false
	uploadParams := uploader.UploadParams{
		PublicID:     publicID,
		Folder:       c.folder,
		ResourceType: "raw", // For non-image/video files
		Overwrite:    &overwrite,
	}

	result, err := c.cld.Upload.Upload(ctx, file, uploadParams)
	if err != nil {
		return nil, fmt.Errorf("failed to upload document: %w", err)
	}

	return &UploadResult{
		URL:          result.SecureURL,
		PublicID:     result.PublicID,
		ResourceType: result.ResourceType,
		Format:       result.Format,
		Bytes:        int64(result.Bytes),
	}, nil
}

// DeleteResource deletes a resource from Cloudinary
func (c *Client) DeleteResource(ctx context.Context, publicID string, resourceType string) error {
	invalidate := true
	params := uploader.DestroyParams{
		PublicID:     publicID,
		ResourceType: resourceType,
		Invalidate:   &invalidate,
	}

	_, err := c.cld.Upload.Destroy(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to delete resource: %w", err)
	}

	return nil
}

// generatePublicID creates a unique public ID for the file
func (c *Client) generatePublicID(filename, subfolder string) string {
	// Remove extension
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	
	// Sanitize filename
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ToLower(name)
	
	// Add timestamp for uniqueness
	timestamp := time.Now().Unix()
	
	return fmt.Sprintf("%s/%s_%d", subfolder, name, timestamp)
}

// ValidateVideoFile checks if the file is a valid video
func ValidateVideoFile(header *multipart.FileHeader) error {
	// Check file size (max 500MB)
	maxSize := int64(500 * 1024 * 1024)
	if header.Size > maxSize {
		return fmt.Errorf("video file too large (max 500MB)")
	}

	// Check content type
	contentType := header.Header.Get("Content-Type")
	validTypes := []string{
		"video/mp4",
		"video/mpeg",
		"video/quicktime",
		"video/x-msvideo",
		"video/x-matroska",
		"video/webm",
	}

	for _, validType := range validTypes {
		if contentType == validType {
			return nil
		}
	}

	return fmt.Errorf("invalid video format (supported: MP4, MPEG, MOV, AVI, MKV, WEBM)")
}

// ValidateDocumentFile checks if the file is a valid document
func ValidateDocumentFile(header *multipart.FileHeader) error {
	// Check file size (max 50MB)
	maxSize := int64(50 * 1024 * 1024)
	if header.Size > maxSize {
		return fmt.Errorf("document file too large (max 50MB)")
	}

	// Check content type
	contentType := header.Header.Get("Content-Type")
	validTypes := []string{
		"application/pdf",
		"application/msword",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/vnd.ms-powerpoint",
		"application/vnd.openxmlformats-officedocument.presentationml.presentation",
		"application/zip",
		"application/x-zip-compressed",
	}

	for _, validType := range validTypes {
		if contentType == validType {
			return nil
		}
	}

	return fmt.Errorf("invalid document format (supported: PDF, DOC, DOCX, PPT, PPTX, ZIP)")
}
