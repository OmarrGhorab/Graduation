package http

import (
	parentApp "github.com/OmarrGhorab/courses-attendance-service/internal/application/parent"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/authclient"
	"github.com/OmarrGhorab/courses-attendance-service/internal/interfaces/http/dto"
	"github.com/OmarrGhorab/courses-attendance-service/internal/interfaces/http/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type ParentHandler struct {
	parentService *parentApp.Service
	authClient    *authclient.Client
}

func NewParentHandler(parentService *parentApp.Service, authClient *authclient.Client) *ParentHandler {
	return &ParentHandler{
		parentService: parentService,
		authClient:    authClient,
	}
}

func (h *ParentHandler) RegisterRoutes(router fiber.Router) {
	auth := middleware.Authenticate(h.authClient)
	parentOnly := middleware.RequireRole("PARENT")

	parent := router.Group("/parent", auth, parentOnly)
	parent.Get("/kids", h.GetChildren)
	parent.Get("/kids/:id/progress", h.GetChildProgress)
	parent.Get("/kids/:id/attendance", h.GetChildAttendance)
}

// GetChildren godoc
// @Summary List all linked children for a parent
// @Tags parent
// @Produce json
// @Success 200 {array} parentApp.ChildSummary
// @Router /api/v1/parent/kids [get]
func (h *ParentHandler) GetChildren(c *fiber.Ctx) error {
	parentID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	children, err := h.parentService.GetChildren(c.Context(), parentID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch children"})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    children,
	})
}

// GetChildProgress godoc
// @Summary Get detailed course progress for a child
// @Tags parent
// @Produce json
// @Param id path string true "Child Student ID"
// @Success 200 {object} parentApp.ChildDetailedProgress
// @Router /api/v1/parent/kids/{id}/progress [get]
func (h *ParentHandler) GetChildProgress(c *fiber.Ctx) error {
	parentID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	studentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid student ID"})
	}

	progress, err := h.parentService.GetChildDetailedProgress(c.Context(), parentID, studentID)
	if err != nil {
		if err == parentApp.ErrParentNotLinked {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "This student is not linked to you"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch progress"})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    progress,
	})
}

// GetChildAttendance godoc
// @Summary Get full attendance history for a child
// @Tags parent
// @Produce json
// @Param id path string true "Child Student ID"
// @Success 200 {array} attendanceDomain.AttendanceRecord
// @Router /api/v1/parent/kids/{id}/attendance [get]
func (h *ParentHandler) GetChildAttendance(c *fiber.Ctx) error {
	parentID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	studentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid student ID"})
	}

	history, err := h.parentService.GetChildAttendanceHistory(c.Context(), parentID, studentID)
	if err != nil {
		if err == parentApp.ErrParentNotLinked {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "This student is not linked to you"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch attendance history"})
	}

	responses := make([]dto.AttendanceRecordResponse, len(history))
	for i, r := range history {
		responses[i] = dto.ToAttendanceRecordResponse(&r.AttendanceRecord)
		responses[i].LessonTitle = r.LessonTitle
		responses[i].CourseTitle = r.CourseTitle
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    responses,
	})
}
