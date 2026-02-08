package http

import (
	progressApp "github.com/OmarrGhorab/courses-attendance-service/internal/application/progress"
	"github.com/OmarrGhorab/courses-attendance-service/internal/interfaces/http/dto"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// ProgressHandler handles progress-related HTTP requests
type ProgressHandler struct {
	progressService *progressApp.Service
}

func NewProgressHandler(progressService *progressApp.Service) *ProgressHandler {
	return &ProgressHandler{progressService: progressService}
}

func (h *ProgressHandler) RegisterRoutes(router fiber.Router) {
	progress := router.Group("/progress")
	progress.Get("/student/:courseId/:studentId", h.GetStudentProgress)
	progress.Get("/course/:courseId", h.GetCourseProgress)
	progress.Post("/recompute/:courseId/:studentId", h.RecomputeProgress)
}

// GetStudentProgress godoc
// @Summary Get latest progress for a student in a course
// @Tags progress
// @Produce json
// @Param courseId path string true "Course ID"
// @Param studentId path string true "Student ID"
// @Success 200 {object} dto.ProgressResponse
// @Router /api/v1/progress/student/{courseId}/{studentId} [get]
func (h *ProgressHandler) GetStudentProgress(c *fiber.Ctx) error {
	courseID, err := uuid.Parse(c.Params("courseId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid course ID"})
	}
	studentID, err := uuid.Parse(c.Params("studentId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid student ID"})
	}

	snapshot, err := h.progressService.GetStudentProgress(c.Context(), courseID, studentID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if snapshot == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Progress not found"})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    dto.ToProgressResponse(snapshot),
	})
}

// GetCourseProgress godoc
// @Summary Get progress for all students in a course
// @Tags progress
// @Produce json
// @Param courseId path string true "Course ID"
// @Success 200 {array} dto.ProgressResponse
// @Router /api/v1/progress/course/{courseId} [get]
func (h *ProgressHandler) GetCourseProgress(c *fiber.Ctx) error {
	courseID, err := uuid.Parse(c.Params("courseId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid course ID"})
	}

	snapshots, err := h.progressService.GetCourseProgress(c.Context(), courseID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	responses := make([]dto.ProgressResponse, len(snapshots))
	for i, s := range snapshots {
		responses[i] = dto.ToProgressResponse(&s)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    responses,
	})
}

// RecomputeProgress godoc
// @Summary Recompute progress for a student in a course
// @Tags progress
// @Produce json
// @Param courseId path string true "Course ID"
// @Param studentId path string true "Student ID"
// @Success 200 {object} dto.ProgressResponse
// @Router /api/v1/progress/recompute/{courseId}/{studentId} [post]
func (h *ProgressHandler) RecomputeProgress(c *fiber.Ctx) error {
	courseID, err := uuid.Parse(c.Params("courseId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid course ID"})
	}
	studentID, err := uuid.Parse(c.Params("studentId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid student ID"})
	}

	snapshot, err := h.progressService.RecomputeProgress(c.Context(), courseID, studentID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    dto.ToProgressResponse(snapshot),
	})
}
