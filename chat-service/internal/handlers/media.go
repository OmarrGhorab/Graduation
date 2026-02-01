package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/graduation/chat-service/internal/services"
)

// MediaHandler handles media upload HTTP requests
type MediaHandler struct {
	mediaSvc *services.MediaService
}

// NewMediaHandler creates a new MediaHandler
func NewMediaHandler(mediaSvc *services.MediaService) *MediaHandler {
	return &MediaHandler{mediaSvc: mediaSvc}
}

// PresignRequest is the request body for presigning upload URL
type PresignRequest struct {
	Type        string `json:"type"`         // "image" or "voice"
	ContentType string `json:"content_type"` // e.g., "image/jpeg"
	FileSize    int64  `json:"file_size"`    // in bytes
}

// Presign generates a presigned URL for media upload
func (h *MediaHandler) Presign(c *fiber.Ctx) error {
	var req PresignRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{"code": "BAD_REQUEST", "message": "Invalid request body"},
		})
	}

	if req.Type == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{"code": "BAD_REQUEST", "message": "Media type is required"},
		})
	}

	if req.FileSize <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{"code": "BAD_REQUEST", "message": "File size must be positive"},
		})
	}

	mediaType := services.MediaType(req.Type)
	if mediaType != services.MediaTypeImage && mediaType != services.MediaTypeVoice {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{"code": "BAD_REQUEST", "message": "Invalid media type, must be 'image' or 'voice'"},
		})
	}

	// Validate size
	if err := h.mediaSvc.ValidateMediaSize(mediaType, req.FileSize); err != nil {
		return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{
			"error": fiber.Map{"code": "PAYLOAD_TOO_LARGE", "message": err.Error()},
		})
	}

	response, err := h.mediaSvc.GeneratePresignedURL(mediaType, req.ContentType, req.FileSize)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{"code": "INTERNAL_ERROR", "message": err.Error()},
		})
	}

	return c.JSON(response)
}
