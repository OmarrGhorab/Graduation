package http

import (
	"github.com/OmarrGhorab/courses-attendance-service/internal/application/course"
	"github.com/OmarrGhorab/courses-attendance-service/internal/interfaces/http/dto"
	"github.com/OmarrGhorab/courses-attendance-service/internal/interfaces/http/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type InternalHandler struct {
	courseService  *course.Service
	internalSecret string
}

func NewInternalHandler(courseService *course.Service, internalSecret string) *InternalHandler {
	return &InternalHandler{
		courseService:  courseService,
		internalSecret: internalSecret,
	}
}

func (h *InternalHandler) RegisterRoutes(router fiber.Router) {
	internal := router.Group("/internal", middleware.InternalOnly(h.internalSecret))

	internal.Get("/courses/:id", h.GetCourse)
	internal.Post("/enrollments/activate", h.ActivateEnrollment)
}

func (h *InternalHandler) GetCourse(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid course ID",
		})
	}

	course, err := h.courseService.GetCourse(c.Context(), id)
	if err != nil {
		return handleServiceError(c, err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    dto.ToCourseResponse(course),
	})
}

func (h *InternalHandler) ActivateEnrollment(c *fiber.Ctx) error {
	var req struct {
		UserID   string `json:"userId"`
		CourseID string `json:"courseId"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid user ID",
		})
	}

	courseID, err := uuid.Parse(req.CourseID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid course ID",
		})
	}

	err = h.courseService.MarkEnrollmentPaid(c.Context(), courseID, userID)
	if err != nil {
		return handleServiceError(c, err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Enrollment activated successfully",
	})
}
