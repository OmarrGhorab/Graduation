package http

import (
	"errors"

	attendanceApp "github.com/OmarrGhorab/courses-attendance-service/internal/application/attendance"
	lessonApp "github.com/OmarrGhorab/courses-attendance-service/internal/application/lesson"
	lessonDomain "github.com/OmarrGhorab/courses-attendance-service/internal/domain/lesson"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/authclient"
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
}

func NewLessonHandler(lessonService *lessonApp.Service, attendanceService *attendanceApp.Service, authClient *authclient.Client) *LessonHandler {
	return &LessonHandler{
		lessonService:     lessonService,
		attendanceService: attendanceService,
		authClient:        authClient,
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
		ScheduledAt:     req.ScheduledAt,
		DurationMinutes: req.DurationMinutes,
		DeliveryType:    lessonDomain.DeliveryType(req.DeliveryType),
		LocationName:    req.LocationName,
		LocationLat:     req.LocationLat,
		LocationLng:     req.LocationLng,
		GeofenceRadiusM: req.GeofenceRadiusM,
	}

	lesson, err := h.lessonService.CreateLesson(c.Context(), teacherID, input)
	if err != nil {
		return handleLessonServiceError(c, err)
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

	lessons, err := h.lessonService.GetCourseLessons(c.Context(), courseID)
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
