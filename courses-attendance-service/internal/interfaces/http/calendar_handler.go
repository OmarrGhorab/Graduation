package http

import (
	"time"

	calendarApp "github.com/OmarrGhorab/courses-attendance-service/internal/application/calendar"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/authclient"
	"github.com/OmarrGhorab/courses-attendance-service/internal/interfaces/http/dto"
	"github.com/OmarrGhorab/courses-attendance-service/internal/interfaces/http/middleware"
	"github.com/gofiber/fiber/v2"
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
// @Success 200 {array} dto.CalendarEventResponse
// @Router /api/v1/calendar/student [get]
func (h *CalendarHandler) GetStudentCalendar(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	start, end := parseTimeRange(c)

	events, err := h.calendarService.GetStudentCalendar(c.Context(), userID, start, end)
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
// @Success 200 {array} dto.CalendarEventResponse
// @Router /api/v1/calendar/teacher [get]
func (h *CalendarHandler) GetTeacherCalendar(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	start, end := parseTimeRange(c)

	events, err := h.calendarService.GetTeacherCalendar(c.Context(), userID, start, end)
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

func parseTimeRange(c *fiber.Ctx) (time.Time, time.Time) {
	startStr := c.Query("start")
	endStr := c.Query("end")

	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		start = time.Now().AddDate(0, 0, -7) // Default 1 week back
	}

	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		end = time.Now().AddDate(0, 1, 0) // Default 1 month ahead
	}

	return start, end
}
