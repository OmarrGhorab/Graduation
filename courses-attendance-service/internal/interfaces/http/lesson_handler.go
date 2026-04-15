package http

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	attendanceApp "github.com/OmarrGhorab/courses-attendance-service/internal/application/attendance"
	lessonApp "github.com/OmarrGhorab/courses-attendance-service/internal/application/lesson"
	lessonDomain "github.com/OmarrGhorab/courses-attendance-service/internal/domain/lesson"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/authclient"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/cloudinary"
	"github.com/OmarrGhorab/courses-attendance-service/internal/interfaces/http/dto"
	"github.com/OmarrGhorab/courses-attendance-service/internal/interfaces/http/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// LessonHandler handles lesson-related HTTP requests
type LessonHandler struct {
	lessonService     *lessonApp.Service
	attendanceService *attendanceApp.Service
	authClient        *authclient.Client
	cloudinaryClient  *cloudinary.Client
}

func NewLessonHandler(lessonService *lessonApp.Service, attendanceService *attendanceApp.Service, authClient *authclient.Client, cloudinaryClient *cloudinary.Client) *LessonHandler {
	return &LessonHandler{
		lessonService:     lessonService,
		attendanceService: attendanceService,
		authClient:        authClient,
		cloudinaryClient:  cloudinaryClient,
	}
}

func (h *LessonHandler) RegisterRoutes(router fiber.Router) {
	auth := middleware.Authenticate(h.authClient)
	teacherOnly := middleware.RequireRole("TEACHER", "INSTRUCTOR")

	lessons := router.Group("/lessons", auth)
	lessons.Post("/", teacherOnly, h.CreateLesson)
	lessons.Get("/:id", h.GetLesson)
	lessons.Post("/:id/start", teacherOnly, h.StartLesson)
	lessons.Post("/:id/end", teacherOnly, h.EndLesson)
	lessons.Post("/:id/cancel", teacherOnly, h.CancelLesson)
	lessons.Post("/:id/reschedule", teacherOnly, h.RescheduleLesson)
	lessons.Put("/:id/materials", teacherOnly, h.UpdateLessonMaterials)
	lessons.Post("/:id/upload-video", teacherOnly, h.UploadVideo)       // NEW: Upload video
	lessons.Post("/:id/upload-document", teacherOnly, h.UploadDocument) // NEW: Upload document
	lessons.Delete("/:id/video", teacherOnly, h.DeleteVideo)            // NEW: Delete video

	// Course lessons
	router.Get("/courses/:id/lessons", auth, h.GetCourseLessons)
}

// CreateLesson godoc
// @Summary Create a new lesson
// @Tags lessons
// @Accept json
// @Produce json
// @Param body body dto.CreateLessonRequest true "Lesson data"
// @Success 201 {object} dto.LessonResponse
// @Router /api/v1/lessons [post]
func (h *LessonHandler) CreateLesson(c *fiber.Ctx) error {
	var req dto.CreateLessonRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	if err := ValidateStruct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"errors":  FormatValidationErrors(err),
		})
	}

	teacherID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "Unauthorized",
		})
	}

	courseID, _ := uuid.Parse(req.CourseID)

	input := lessonApp.CreateLessonInput{
		CourseID:        courseID,
		Title:           req.Title,
		Description:     req.Description,
		ThumbnailURL:    req.ThumbnailURL,
		ScheduledAt:     req.ScheduledAt,
		DurationMinutes: req.DurationMinutes,
		DeliveryType:    lessonDomain.DeliveryType(req.DeliveryType),
		IsFree:          req.IsFree,
		VideoURL:        req.VideoURL,
		VideoPublicID:   req.VideoPublicID,
		MaterialsURL:    req.MaterialsURL,
		Duration:        req.Duration,
		LocationName:    req.LocationName,
		LocationLat:     req.LocationLat,
		LocationLng:     req.LocationLng,
		GeofenceRadiusM: req.GeofenceRadiusM,
	}

	lesson, err := h.lessonService.CreateLesson(c.Context(), teacherID, input)
	if err != nil {
		return handleLessonServiceError(c, err)
	}

	// NEW: Check if a video file is attached in the creation request (Asynchronous Processing)
	if req.DeliveryType == "ONLINE" {
		fileHeader, err := c.FormFile("video")
		if err == nil && fileHeader != nil {
			// 1. Create a persistent temporary file to store the video for background processing
			tempDir := os.TempDir()
			tempFileName := fmt.Sprintf("upload_%s_%d%s", lesson.ID.String(), time.Now().Unix(), filepath.Ext(fileHeader.Filename))
			tempPath := filepath.Join(tempDir, tempFileName)

			// 2. Open source file
			src, err := fileHeader.Open()
			if err == nil {
				defer src.Close()
				
				// 3. Create destination temp file
				dst, err := os.Create(tempPath)
				if err == nil {
					defer dst.Close()
					
					// 4. Copy data to temp file
					if _, err = io.Copy(dst, src); err == nil {
						// 5. Start background processing
						// We pass a background context because the request context will be canceled
						go h.lessonService.ProcessLessonVideoAsync(context.Background(), lesson.ID, teacherID, tempPath, fileHeader.Filename)
					}
				}
			}
		}
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    dto.ToLessonResponse(lesson),
	})
}

// GetLesson godoc
// @Summary Get a lesson by ID
// @Tags lessons
// @Produce json
// @Param id path string true "Lesson ID"
// @Success 200 {object} dto.LessonResponse
// @Router /api/v1/lessons/{id} [get]
func (h *LessonHandler) GetLesson(c *fiber.Ctx) error {
	lessonID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid lesson ID",
		})
	}

	lesson, err := h.lessonService.GetLesson(c.Context(), lessonID)
	if err != nil {
		return handleLessonServiceError(c, err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    dto.ToLessonResponse(lesson),
	})
}

// GetCourseLessons godoc
// @Summary Get all lessons for a course
// @Tags lessons
// @Produce json
// @Param id path string true "Course ID"
// @Success 200 {array} dto.LessonResponse
// @Router /api/v1/courses/{id}/lessons [get]
func (h *LessonHandler) GetCourseLessons(c *fiber.Ctx) error {
	courseID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid course ID",
		})
	}

	userID, _ := getUserIDFromContext(c)
	userRole := c.Locals("userRole").(string)

	lessons, err := h.lessonService.GetCourseLessons(c.Context(), courseID, userID, userRole)


	if err != nil {
		return handleLessonServiceError(c, err)
	}

	var responses []dto.LessonResponse
	for _, l := range lessons {
		responses = append(responses, dto.ToLessonResponse(&l))
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    responses,
	})
}

// StartLesson godoc
// @Summary Start a lesson (sets status to LIVE and creates attendance session)
// @Tags lessons
// @Produce json
// @Param id path string true "Lesson ID"
// @Success 200 {object} dto.LessonResponse
// @Router /api/v1/lessons/{id}/start [post]
func (h *LessonHandler) StartLesson(c *fiber.Ctx) error {
	lessonID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid lesson ID",
		})
	}

	// Start the lesson (sets status to LIVE)
	lesson, err := h.lessonService.StartLesson(c.Context(), lessonID)
	if err != nil {
		return handleLessonServiceError(c, err)
	}

	// Automatically start attendance session and generate first QR token
	session, err := h.attendanceService.StartAttendanceSession(c.Context(), lessonID)
	if err != nil {
		// Log error but don't fail the request - lesson is already started
		// In production, you might want to handle this differently
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Lesson started but failed to create attendance session",
		})
	}

	// Get the first QR token
	qrToken, err := h.attendanceService.GetCurrentQRToken(c.Context(), lessonID)
	if err != nil {
		// QR token generation failed, but session is created
		return c.JSON(fiber.Map{
			"success": true,
			"data":    dto.ToLessonResponse(lesson),
			"message": "Lesson and attendance session started successfully, but QR generation failed",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    dto.ToLessonResponse(lesson),
		"message": "Lesson started successfully",
		"attendance_session": fiber.Map{
			"session_id": session.ID,
			"started_at": session.StartedAt,
			"is_active":  session.IsActive,
		},
		"qr_token": fiber.Map{
			"payload":    qrToken.Payload,
			"signature":  qrToken.Signature,
			"issued_at":  qrToken.IssuedAt,
			"expires_at": qrToken.ExpiresAt,
		},
	})
}

// EndLesson godoc
// @Summary End a lesson (sets status to COMPLETED and ends attendance session)
// @Tags lessons
// @Produce json
// @Param id path string true "Lesson ID"
// @Success 200 {object} dto.LessonResponse
// @Router /api/v1/lessons/{id}/end [post]
func (h *LessonHandler) EndLesson(c *fiber.Ctx) error {
	lessonID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid lesson ID",
		})
	}

	// End attendance session first (marks absentees)
	if err := h.attendanceService.EndAttendanceSession(c.Context(), lessonID); err != nil {
		// Log error but continue to end lesson
		// In production, you might want to handle this differently
	}

	// End the lesson (sets status to COMPLETED)
	lesson, err := h.lessonService.EndLesson(c.Context(), lessonID)
	if err != nil {
		return handleLessonServiceError(c, err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    dto.ToLessonResponse(lesson),
		"message": "Lesson ended successfully",
	})
}

// CancelLesson godoc
// @Summary Cancel a lesson
// @Tags lessons
// @Produce json
// @Param id path string true "Lesson ID"
// @Success 200 {object} dto.LessonResponse
// @Router /api/v1/lessons/{id}/cancel [post]
func (h *LessonHandler) CancelLesson(c *fiber.Ctx) error {
	lessonID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid lesson ID",
		})
	}

	lesson, err := h.lessonService.CancelLesson(c.Context(), lessonID)
	if err != nil {
		return handleLessonServiceError(c, err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    dto.ToLessonResponse(lesson),
		"message": "Lesson canceled successfully",
	})
}

// RescheduleLesson godoc
// @Summary Reschedule a lesson to a new time
// @Tags lessons
// @Accept json
// @Produce json
// @Param id path string true "Lesson ID"
// @Param body body dto.RescheduleLessonRequest true "New schedule"
// @Success 200 {object} dto.LessonResponse
// @Router /api/v1/lessons/{id}/reschedule [post]
func (h *LessonHandler) RescheduleLesson(c *fiber.Ctx) error {
	lessonID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid lesson ID",
		})
	}

	var req dto.RescheduleLessonRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	if err := ValidateStruct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"errors":  FormatValidationErrors(err),
		})
	}

	lesson, err := h.lessonService.RescheduleLesson(c.Context(), lessonID, req.ScheduledAt)
	if err != nil {
		return handleLessonServiceError(c, err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    dto.ToLessonResponse(lesson),
		"message": "Lesson rescheduled successfully",
	})
}

func handleLessonServiceError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, lessonApp.ErrLessonNotFound):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "Lesson not found",
		})
	case errors.Is(err, lessonApp.ErrCourseNotFound):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "Course not found",
		})
	case errors.Is(err, lessonApp.ErrUnauthorized):
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error":   "You are not authorized to perform this action",
		})
	case errors.Is(err, lessonApp.ErrInvalidStatus):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid lesson status for this operation",
		})
	case errors.Is(err, lessonApp.ErrLessonInProgress):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Cannot perform this action while lesson is in progress",
		})
	case errors.Is(err, lessonApp.ErrLessonNotLive):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Lesson is not currently live",
		})
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Internal server error",
		})
	}
}


// UpdateLessonMaterials godoc
// @Summary Update lesson materials (video, documents)
// @Tags lessons
// @Accept json
// @Produce json
// @Param id path string true "Lesson ID"
// @Param body body dto.UpdateLessonMaterialsRequest true "Materials data"
// @Success 200 {object} dto.LessonResponse
// @Router /api/v1/lessons/{id}/materials [put]
func (h *LessonHandler) UpdateLessonMaterials(c *fiber.Ctx) error {
	lessonID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid lesson ID",
		})
	}

	var req dto.UpdateLessonMaterialsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	// Get lesson
	lesson, err := h.lessonService.GetLesson(c.Context(), lessonID)
	if err != nil {
		return handleLessonServiceError(c, err)
	}

	// Check if lesson is ONLINE
	if lesson.DeliveryType != lessonDomain.DeliveryTypeOnline {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Materials can only be uploaded for ONLINE lessons",
		})
	}

	// Update materials
	if req.VideoURL != nil {
		lesson.VideoURL = *req.VideoURL
	}
	if req.VideoPublicID != nil {
		lesson.VideoPublicID = *req.VideoPublicID
	}
	if req.MaterialsURL != nil {
		lesson.MaterialsURL = *req.MaterialsURL
	}
	if req.Duration != nil {
		lesson.Duration = req.Duration
	}

	// Save lesson
	if err := h.lessonService.UpdateLesson(c.Context(), lesson); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to update lesson materials",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    dto.ToLessonResponse(lesson),
		"message": "Lesson materials updated successfully",
	})
}


// UploadVideo godoc
// @Summary Upload video file to Cloudinary for a lesson
// @Tags lessons
// @Accept multipart/form-data
// @Produce json
// @Param id path string true "Lesson ID"
// @Param video formData file true "Video file"
// @Success 200 {object} dto.LessonResponse
// @Router /api/v1/lessons/{id}/upload-video [post]
func (h *LessonHandler) UploadVideo(c *fiber.Ctx) error {
	lessonID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid lesson ID",
		})
	}

	// Get lesson
	lesson, err := h.lessonService.GetLesson(c.Context(), lessonID)
	if err != nil {
		return handleLessonServiceError(c, err)
	}

	// Check if lesson is ONLINE
	if lesson.DeliveryType != lessonDomain.DeliveryTypeOnline {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Videos can only be uploaded for ONLINE lessons",
		})
	}

	// Get file from form
	fileHeader, err := c.FormFile("video")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Video file is required",
		})
	}

	// Validate video file
	if err := cloudinary.ValidateVideoFile(fileHeader); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	// Open file
	file, err := fileHeader.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to read video file",
		})
	}
	defer file.Close()

	// Delete old video if exists
	if lesson.VideoPublicID != "" {
		_ = h.cloudinaryClient.DeleteResource(c.Context(), lesson.VideoPublicID, "video")
	}

	// Upload to Cloudinary
	result, err := h.cloudinaryClient.UploadVideo(c.Context(), file, fileHeader.Filename)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to upload video to Cloudinary",
		})
	}

	// Update lesson
	lesson.VideoURL = result.StreamingURL
	lesson.VideoPublicID = result.PublicID
	lesson.Duration = result.Duration

	if err := h.lessonService.UpdateLesson(c.Context(), lesson); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to update lesson",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    dto.ToLessonResponse(lesson),
		"message": "Video uploaded successfully",
		"upload_info": fiber.Map{
			"size_mb":  float64(result.Bytes) / (1024 * 1024),
			"duration": result.Duration,
			"format":   result.Format,
		},
	})
}

// UploadDocument godoc
// @Summary Upload document file to Cloudinary for a lesson
// @Tags lessons
// @Accept multipart/form-data
// @Produce json
// @Param id path string true "Lesson ID"
// @Param document formData file true "Document file (PDF, DOC, PPT, etc.)"
// @Success 200 {object} dto.LessonResponse
// @Router /api/v1/lessons/{id}/upload-document [post]
func (h *LessonHandler) UploadDocument(c *fiber.Ctx) error {
	lessonID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid lesson ID",
		})
	}

	// Get lesson
	lesson, err := h.lessonService.GetLesson(c.Context(), lessonID)
	if err != nil {
		return handleLessonServiceError(c, err)
	}

	// Check if lesson is ONLINE
	if lesson.DeliveryType != lessonDomain.DeliveryTypeOnline {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Documents can only be uploaded for ONLINE lessons",
		})
	}

	// Get file from form
	fileHeader, err := c.FormFile("document")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Document file is required",
		})
	}

	// Validate document file
	if err := cloudinary.ValidateDocumentFile(fileHeader); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	// Open file
	file, err := fileHeader.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to read document file",
		})
	}
	defer file.Close()

	// Upload to Cloudinary
	result, err := h.cloudinaryClient.UploadDocument(c.Context(), file, fileHeader.Filename)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to upload document to Cloudinary",
		})
	}

	// Update lesson
	lesson.MaterialsURL = result.URL

	if err := h.lessonService.UpdateLesson(c.Context(), lesson); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to update lesson",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    dto.ToLessonResponse(lesson),
		"message": "Document uploaded successfully",
		"upload_info": fiber.Map{
			"size_mb": float64(result.Bytes) / (1024 * 1024),
			"format":  result.Format,
		},
	})
}

// DeleteVideo godoc
// @Summary Delete video from Cloudinary and lesson
// @Tags lessons
// @Produce json
// @Param id path string true "Lesson ID"
// @Success 200 {object} dto.LessonResponse
// @Router /api/v1/lessons/{id}/video [delete]
func (h *LessonHandler) DeleteVideo(c *fiber.Ctx) error {
	lessonID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid lesson ID",
		})
	}

	// Get lesson
	lesson, err := h.lessonService.GetLesson(c.Context(), lessonID)
	if err != nil {
		return handleLessonServiceError(c, err)
	}

	if lesson.VideoPublicID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "No video to delete",
		})
	}

	// Delete from Cloudinary
	if err := h.cloudinaryClient.DeleteResource(c.Context(), lesson.VideoPublicID, "video"); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to delete video from Cloudinary",
		})
	}

	// Update lesson
	lesson.VideoURL = ""
	lesson.VideoPublicID = ""
	lesson.Duration = nil

	if err := h.lessonService.UpdateLesson(c.Context(), lesson); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to update lesson",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    dto.ToLessonResponse(lesson),
		"message": "Video deleted successfully",
	})
}
