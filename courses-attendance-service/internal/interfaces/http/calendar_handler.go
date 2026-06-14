package http

import (
	"strings"
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
	parentOnly := middleware.RequireRole("PARENT")

	calendar := router.Group("/calendar", auth)
	calendar.Get("/student", h.GetStudentCalendar)
	calendar.Get("/teacher", managementOnly, h.GetTeacherCalendar)
	calendar.Get("/parent", parentOnly, h.GetParentCalendar)
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

	events, total, err := h.calendarService.GetStudentCalendar(c.Context(), userID, filter)
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
		"meta": fiber.Map{
			"page":       filter.Page,
			"limit":      filter.Limit,
			"total":      total,
			"totalPages": totalPages(total, filter.Limit),
		},
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

	events, total, err := h.calendarService.GetTeacherCalendar(c.Context(), userID, filter)
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
		"meta": fiber.Map{
			"page":       filter.Page,
			"limit":      filter.Limit,
			"total":      total,
			"totalPages": totalPages(total, filter.Limit),
		},
	})
}

// GetParentCalendar returns calendar events for parent's children.
// Optional ?child_id=UUID to scope to a single child.
func (h *CalendarHandler) GetParentCalendar(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	filter := h.parseCalendarFilter(c)

	// Fetch all linked children to verify ownership
	children, err := h.authClient.GetChildren(c.Context(), userID.String())
	if err != nil || len(children) == 0 {
		return c.JSON(fiber.Map{
			"success": true,
			"data":    []dto.CalendarEventResponse{},
			"meta":    fiber.Map{"page": filter.Page, "limit": filter.Limit, "total": 0, "totalPages": 0},
		})
	}

	// Build full map of linked children
	allChildIDs := make([]uuid.UUID, 0, len(children))
	childNameMap := make(map[uuid.UUID]string, len(children))
	for _, ch := range children {
		if uid, err := uuid.Parse(ch.ID); err == nil {
			allChildIDs = append(allChildIDs, uid)
			childNameMap[uid] = ch.Name
		}
	}

	// If a specific child_id was requested, validate it belongs to this parent
	childIDs := allChildIDs
	if childIDStr := c.Query("child_id"); childIDStr != "" {
		if requestedUID, err := uuid.Parse(childIDStr); err == nil {
			// Only allow if this child is actually linked to the parent
			owned := false
			for _, cid := range allChildIDs {
				if cid == requestedUID {
					owned = true
					break
				}
			}
			if owned {
				childIDs = []uuid.UUID{requestedUID}
			}
		}
	}

	events, total, err := h.calendarService.GetParentCalendar(c.Context(), childIDs, childNameMap, filter)
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
		"meta": fiber.Map{
			"page":       filter.Page,
			"limit":      filter.Limit,
			"total":      total,
			"totalPages": totalPages(total, filter.Limit),
		},
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
		case "cancelled":
			statuses = []string{"CANCELED"}
		default:
			parts := strings.Split(statusStr, ",")
			statuses = make([]string, 0, len(parts))
			for _, part := range parts {
				normalized := strings.TrimSpace(strings.ToUpper(part))
				if normalized == "" {
					continue
				}
				if normalized == "CANCELLED" {
					normalized = "CANCELED"
				}
				statuses = append(statuses, normalized)
			}
		}
	}

	page := c.QueryInt("page", 1)
	if page < 1 {
		page = 1
	}

	limit := c.QueryInt("limit", 20)
	if limit < 1 || limit > 100 {
		limit = 20
	}

	return calendarApp.CalendarFilter{
		Start:       start,
		End:         end,
		SubjectID:   subjectID,
		SubjectName: subjectName,
		Statuses:    statuses,
		Page:        page,
		Limit:       limit,
	}
}

func totalPages(total int64, limit int) int64 {
	if limit <= 0 {
		return 1
	}
	if total == 0 {
		return 0
	}
	return (total + int64(limit) - 1) / int64(limit)
}
