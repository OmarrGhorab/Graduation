package service

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"github.com/graduation/chat-service/internal/config"
)

type MediaService struct {
	config *config.Config
}

func NewMediaService(cfg *config.Config) *MediaService {
	return &MediaService{config: cfg}
}

// GeneratePresignedURL generates a signature for Cloudinary upload
// Returns: timestamp, signature, api_key, cloud_name
func (s *MediaService) GeneratePresignedURL(folder string) map[string]string {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	// Signature params: folder, timestamp, source=uw (upstream widget? usually just params sorted)
	// Simple signature: timestamp=...&folder=... + secret

	params := fmt.Sprintf("folder=%s&timestamp=%s%s", folder, timestamp, s.config.CloudinaryAPISecret)

	hash := sha1.New()
	hash.Write([]byte(params))
	signature := hex.EncodeToString(hash.Sum(nil))

	return map[string]string{
		"cloud_name": s.config.CloudinaryCloudName,
		"api_key":    s.config.CloudinaryAPIKey,
		"timestamp":  timestamp,
		"folder":     folder,
		"signature":  signature,
	}
}
