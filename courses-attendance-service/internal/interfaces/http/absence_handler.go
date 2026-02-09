package http

import (
	"errors"

	absenceApp "github.com/OmarrGhorab/courses-attendance-service/internal/application/absence"
	absenceDomain "github.com/OmarrGhorab/courses-attendance-service/internal/domain/absence"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/authclient"
	"github.com/OmarrGhorab/courses-attendance-service/internal/interfaces/http/dto"
	"github.com/OmarrGhorab/courses-attendance-service/internal/interfaces/http/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// AbsenceHandler handles absence-related HTTP requests
type AbsenceHandler struct {
	absenceService *absenceApp.Service
	authClient     *authclient.Client
}

func NewAbsenceHandler(absenceService *absenceApp.Service, authClient *authclient.Client) *AbsenceHandler {
	return &AbsenceHandler{
		absenceService: absenceService,
		authClient:     authClient,
	}
}

func (h *AbsenceHandler) RegisterRoutes(router fiber.Router) {
	auth := middleware.Authenticate(h.authClient)
	managementOnly := middleware.RequireRole("TEACHER", "INSTRUCTOR", "ASSISTANT")

	absences := router.Group("/absences", auth)
	absences.Post("/", h.CreateRequest)
	absences.Get("/student/:id", h.GetStudentRequests)
	absences.Get("/lesson/:id", managementOnly, h.GetLessonRequests)
	absences.Get("/pending-parent", h.GetPendingParentRequests)
	absences.Post("/:id/respond", h.RespondToRequest)
}

// CreateRequest godoc
// @Summary Create an absence request
// @Tags absences
// @Accept json
// @Produce json
// @Param body body dto.CreateAbsenceRequest true "Request data"
// @Success 201 {object} dto.AbsenceRequestResponse
// @Router /api/v1/absences [post]
func (h *AbsenceHandler) CreateRequest(c *fiber.Ctx) error {
	var req dto.CreateAbsenceRequest
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

	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "Unauthorized",
		})
	}

	lessonID, _ := uuid.Parse(req.LessonID)
	studentID, _ := uuid.Parse(req.StudentID)

	input := absenceApp.CreateRequestInput{
		LessonID:    lessonID,
		StudentID:   studentID,
		ReasonType:  absenceDomain.AbsenceReasonType(req.ReasonType),
		ReasonText:  req.ReasonText,
		Attachment:  req.Attachment,
		RequestedBy: userID,
	}

	absenceReq, err := h.absenceService.CreateRequest(c.Context(), input)
	if err != nil {
		return handleAbsenceError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    dto.ToAbsenceRequestResponse(absenceReq),
	})
}

// RespondToRequest godoc
// @Summary Approve or reject an absence request
// @Tags absences
// @Accept json
// @Produce json
// @Param id path string true "Request ID"
// @Param body body dto.RespondAbsenceRequest true "Response data"
// @Success 200 {object} dto.AbsenceRequestResponse
// @Router /api/v1/absences/{id}/respond [post]
func (h *AbsenceHandler) RespondToRequest(c *fiber.Ctx) error {
	requestID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request ID",
		})
	}

	var req dto.RespondAbsenceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "Unauthorized",
		})
	}

	input := absenceApp.RespondRequestInput{
		RequestID:    requestID,
		RespondedBy:  userID,
		Approve:      req.Approve,
		ResponseNote: req.ResponseNote,
	}

	absenceReq, err := h.absenceService.RespondToRequest(c.Context(), input)
	if err != nil {
		return handleAbsenceError(c, err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    dto.ToAbsenceRequestResponse(absenceReq),
	})
}

// GetStudentRequests godoc
// @Summary Get all absence requests for a student
// @Tags absences
// @Produce json
// @Param id path string true "Student ID"
// @Success 200 {array} dto.AbsenceRequestResponse
// @Router /api/v1/absences/student/{id} [get]
func (h *AbsenceHandler) GetStudentRequests(c *fiber.Ctx) error {
	studentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid student ID",
		})
	}

	requests, err := h.absenceService.GetStudentRequests(c.Context(), studentID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to fetch absence requests",
		})
	}

	var responses []dto.AbsenceRequestResponse
	for _, r := range requests {
		responses = append(responses, dto.ToAbsenceRequestResponse(&r))
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    responses,
	})
}

// GetLessonRequests godoc
// @Summary Get all absence requests for a lesson
// @Tags absences
// @Produce json
// @Param id path string true "Lesson ID"
// @Success 200 {array} dto.AbsenceRequestResponse
// @Router /api/v1/absences/lesson/{id} [get]
func (h *AbsenceHandler) GetLessonRequests(c *fiber.Ctx) error {
	lessonID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid lesson ID",
		})
	}

	requests, err := h.absenceService.GetLessonRequests(c.Context(), lessonID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to fetch absence requests",
		})
	}

	var responses []dto.AbsenceRequestResponse
	for _, r := range requests {
		responses = append(responses, dto.ToAbsenceRequestResponse(&r))
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    responses,
	})
}

// GetPendingParentRequests godoc
// @Summary Get all pending requests for students linked to this parent
// @Tags absences
// @Produce json
// @Success 200 {array} dto.AbsenceRequestResponse
// @Router /api/v1/absences/pending-parent [get]
func (h *AbsenceHandler) GetPendingParentRequests(c *fiber.Ctx) error {
	parentID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "Unauthorized",
		})
	}

	requests, err := h.absenceService.GetPendingParentRequests(c.Context(), parentID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to fetch absence requests",
		})
	}

	var responses []dto.AbsenceRequestResponse
	for _, r := range requests {
		responses = append(responses, dto.ToAbsenceRequestResponse(&r))
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    responses,
	})
}

func handleAbsenceError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, absenceApp.ErrRequestNotFound):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "Absence request not found",
		})
	case errors.Is(err, absenceApp.ErrUnauthorized):
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error":   "You are not authorized to perform this action",
		})
	case errors.Is(err, absenceApp.ErrParentNotLinked):
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error":   "User is not linked to this student as a parent",
		})
	case errors.Is(err, absenceApp.ErrInvalidStatus):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "This request is already processed",
		})
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Internal server error",
		})
	}
}
