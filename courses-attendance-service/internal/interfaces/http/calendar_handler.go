package http

import (
	"time"

	calendarApp "github.com/OmarrGhorab/courses-attendance-service/internal/application/calendar"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/authclient"
	"github.com/OmarrGhorab/courses-attendance-service/internal/interfaces/http/dto"
	"github.com/OmarrGhorab/courses-attendance-service/internal/interfaces/http/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// CalendarHandler handles calendar-related HTTP requests
type CalendarHandler struct {
	calendarService *calendarApp.Service
	authClient      *authclient.Client
}

func NewCalendarHandler(calendarService *calendarApp.Service, authClient *authclient.Client) *CalendarHandler {
	return &CalendarHandler{
		calendarService: calendarService,
		authClient:      authClient,
	}
}

func (h *CalendarHandler) RegisterRoutes(router fiber.Router) {
	auth := middleware.Authenticate(h.authClient)
	managementOnly := middleware.RequireRole("TEACHER", "INSTRUCTOR", "ASSISTANT")

	calendar := router.Group("/calendar", auth)
	calendar.Get("/student", h.GetStudentCalendar)
	calendar.Get("/teacher", managementOnly, h.GetTeacherCalendar)
}

// GetStudentCalendar godoc
// @Summary Get upcoming lessons for the current student
// @Tags calendar
// @Produce json
// @Param start query string false "Start date (RFC3339)"
// @Param end query string false "End date (RFC3339)"
// @Param range query string false "Preset range (prev_7, upcoming_7, upcoming_30)"
// @Param subject_name query string false "Filter by subject name (e.g. Math)"
// @Param subject query string false "Filter by subject ID (UUID)"
// @Param status query string false "Lesson status (upcoming, finished, or standard: SCHEDULED, LIVE, COMPLETED, CANCELED)"
// @Success 200 {array} dto.CalendarEventResponse
// @Router /api/v1/calendar/student [get]
func (h *CalendarHandler) GetStudentCalendar(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	filter := h.parseCalendarFilter(c)

	events, err := h.calendarService.GetStudentCalendar(c.Context(), userID, filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	responses := make([]dto.CalendarEventResponse, len(events))
	for i, e := range events {
		responses[i] = dto.CalendarEventResponse(e)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    responses,
	})
}

// GetTeacherCalendar godoc
// @Summary Get scheduled lessons for the current teacher
// @Tags calendar
// @Produce json
// @Param start query string false "Start date (RFC3339)"
// @Param end query string false "End date (RFC3339)"
// @Param subject query string false "Subject ID (UUID)"
// @Param status query string false "Lesson status (comma-separated: SCHEDULED,LIVE,COMPLETED,CANCELED)"
// @Success 200 {array} dto.CalendarEventResponse
// @Router /api/v1/calendar/teacher [get]
func (h *CalendarHandler) GetTeacherCalendar(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	filter := h.parseCalendarFilter(c)

	events, err := h.calendarService.GetTeacherCalendar(c.Context(), userID, filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	responses := make([]dto.CalendarEventResponse, len(events))
	for i, e := range events {
		responses[i] = dto.CalendarEventResponse(e)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    responses,
	})
}

func (h *CalendarHandler) parseCalendarFilter(c *fiber.Ctx) calendarApp.CalendarFilter {
	startStr := c.Query("start")
	endStr := c.Query("end")
	rangeStr := c.Query("range")
	subjectName := c.Query("subject_name")

	var start, end time.Time
	now := time.Now()

	// Handle range presets
	if rangeStr != "" {
		switch rangeStr {
		case "prev_7":
			start = now.AddDate(0, 0, -7)
			end = now
		case "upcoming_7":
			start = now
			end = now.AddDate(0, 0, 7)
		case "upcoming_30":
			start = now
			end = now.AddDate(0, 0, 30)
		}
	}

	// Override with explicit start/end if provided
	if start.IsZero() {
		if s, err := time.Parse(time.RFC3339, startStr); err == nil {
			start = s
		} else {
			start = now.AddDate(0, 0, -30) // Default 1 month back
		}
	}

	if end.IsZero() {
		if e, err := time.Parse(time.RFC3339, endStr); err == nil {
			end = e
		} else {
			end = now.AddDate(0, 3, 0) // Default 3 months ahead
		}
	}

	var subjectID *uuid.UUID
	idStr := c.Query("subject")
	if idStr != "" {
		if uid, err := uuid.Parse(idStr); err == nil {
			subjectID = &uid
		}
	}

	// Parsing statuses: comma-separated or keywords
	statusStr := c.Query("status")
	var statuses []string
	if statusStr != "" {
		switch statusStr {
		case "upcoming":
			statuses = []string{"SCHEDULED", "LIVE"}
		case "finished":
			statuses = []string{"COMPLETED"}
		default:
			statuses = []string{statusStr}
		}
	}

	return calendarApp.CalendarFilter{
		Start:       start,
		End:         end,
		SubjectID:   subjectID,
		SubjectName: subjectName,
		Statuses:    statuses,
	}
}
