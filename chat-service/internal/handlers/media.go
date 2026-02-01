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

// BatchPresignRequest is the request body for batch presigning
type BatchPresignRequest struct {
	Files []PresignRequest `json:"files"`
}

// BatchPresign generates multiple presigned URLs and validates total size
func (h *MediaHandler) BatchPresign(c *fiber.Ctx) error {
	var req BatchPresignRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{"code": "BAD_REQUEST", "message": "Invalid request body"},
		})
	}

	if len(req.Files) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{"code": "BAD_REQUEST", "message": "Files list is empty"},
		})
	}

	// Prepare requests for service
	var serviceReqs []struct {
		Type        services.MediaType
		ContentType string
		FileSize    int64
	}

	for _, file := range req.Files {
		if file.Type == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fiber.Map{"code": "BAD_REQUEST", "message": "Media type is required for all files"},
			})
		}
		if file.FileSize <= 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fiber.Map{"code": "BAD_REQUEST", "message": "File size must be positive"},
			})
		}

		mediaType := services.MediaType(file.Type)
		if mediaType != services.MediaTypeImage && mediaType != services.MediaTypeVoice {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fiber.Map{"code": "BAD_REQUEST", "message": "Invalid media type: " + file.Type},
			})
		}

		serviceReqs = append(serviceReqs, struct {
			Type        services.MediaType
			ContentType string
			FileSize    int64
		}{
			Type:        mediaType,
			ContentType: file.ContentType,
			FileSize:    file.FileSize,
		})
	}

	responses, err := h.mediaSvc.GenerateBatchPresignedURLs(serviceReqs)
	if err != nil {
		// Differentiate between user errors (validation) and internal errors
		// For simplicity, we assume validation errors if the error message contains specific keywords
		// Ideally, use custom error types
		return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{
			"error": fiber.Map{"code": "PAYLOAD_TOO_LARGE", "message": err.Error()},
		})
	}

	return c.JSON(fiber.Map{"files": responses})
}
