package http

import (
	"errors"

	attendanceApp "github.com/OmarrGhorab/courses-attendance-service/internal/application/attendance"
	attendanceDomain "github.com/OmarrGhorab/courses-attendance-service/internal/domain/attendance"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/authclient"
	"github.com/OmarrGhorab/courses-attendance-service/internal/interfaces/http/dto"
	"github.com/OmarrGhorab/courses-attendance-service/internal/interfaces/http/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// AttendanceHandler handles attendance-related HTTP requests
type AttendanceHandler struct {
	attendanceService *attendanceApp.Service
	authClient        *authclient.Client
}

func NewAttendanceHandler(attendanceService *attendanceApp.Service, authClient *authclient.Client) *AttendanceHandler {
	return &AttendanceHandler{
		attendanceService: attendanceService,
		authClient:        authClient,
	}
}

func (h *AttendanceHandler) RegisterRoutes(router fiber.Router) {
	auth := middleware.Authenticate(h.authClient)
	managementOnly := middleware.RequireRole("TEACHER", "INSTRUCTOR", "ASSISTANT")

	attendance := router.Group("/attendance", auth)
	attendance.Post("/scan", h.ScanAttendance) // Everyone authenticated can scan (logic handles enrollment)
	attendance.Get("/lesson/:id", managementOnly, h.GetLessonAttendance)
	attendance.Get("/student/:id", managementOnly, h.GetStudentAttendance)
	attendance.Get("/student/:studentId/course/:courseId/analytics", managementOnly, h.GetStudentCourseAnalytics)

	// Lesson QR management
	router.Get("/lessons/:id/qr", auth, managementOnly, h.GetCurrentQR)
	router.Post("/lessons/:id/qr/rotate", auth, managementOnly, h.RotateQR)

	// Manual Override
	attendance.Post("/lesson/:id/override", managementOnly, h.ManualOverride)
}

// ScanAttendance godoc
// @Summary Scan QR code for attendance
// @Tags attendance
// @Accept json
// @Produce json
// @Param body body dto.ScanAttendanceRequest true "Scan data"
// @Success 200 {object} dto.ScanAttendanceResponse
// @Router /api/v1/attendance/scan [post]
func (h *AttendanceHandler) ScanAttendance(c *fiber.Ctx) error {
	var req dto.ScanAttendanceRequest
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

	studentID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "Unauthorized",
		})
	}

	// Get request context
	accessToken := c.Get("Authorization")
	ip := c.IP()
	userAgent := c.Get("User-Agent")

	input := attendanceApp.ScanInput{
		QRPayload:         req.QRPayload,
		QRSignature:       req.QRSignature,
		StudentID:         studentID,
		DeviceID:          req.DeviceID,
		DeviceFingerprint: req.DeviceFingerprint,
		AttestationToken:  req.AttestationToken,
		IP:                ip,
		UserAgent:         userAgent,
		Latitude:          req.Latitude,
		Longitude:         req.Longitude,
		AccessToken:       accessToken,
	}

	result, err := h.attendanceService.ScanAttendance(c.Context(), input)
	if err != nil {
		return handleAttendanceError(c, err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": dto.ScanAttendanceResponse{
			Status:    string(result.Status),
			ScannedAt: result.ScannedAt,
			Distance:  result.Distance,
			Message:   result.Message,
		},
	})
}

// GetLessonAttendance godoc
// @Summary Get attendance records for a lesson
// @Tags attendance
// @Produce json
// @Param id path string true "Lesson ID"
// @Success 200 {array} dto.AttendanceRecordResponse
// @Router /api/v1/attendance/lesson/{id} [get]
func (h *AttendanceHandler) GetLessonAttendance(c *fiber.Ctx) error {
	lessonID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid lesson ID",
		})
	}

	records, err := h.attendanceService.GetLessonAttendance(c.Context(), lessonID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to fetch attendance records",
		})
	}

	var responses []dto.AttendanceRecordResponse
	for _, r := range records {
		response := dto.ToAttendanceRecordResponse(&r)
		
		// Fetch student info
		userInfo, err := h.authClient.GetUserInfo(c.Context(), r.StudentID.String())
		if err == nil && userInfo != nil {
			response.StudentName = userInfo.Name
			response.StudentProfileImg = userInfo.ProfileImg
		}
		
		responses = append(responses, response)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    responses,
	})
}

// GetStudentAttendance godoc
// @Summary Get attendance records for a student
// @Tags attendance
// @Produce json
// @Param id path string true "Student ID"
// @Success 200 {array} dto.AttendanceRecordResponse
// @Router /api/v1/attendance/student/{id} [get]
func (h *AttendanceHandler) GetStudentAttendance(c *fiber.Ctx) error {
	studentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid student ID",
		})
	}

	records, err := h.attendanceService.GetStudentAttendance(c.Context(), studentID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to fetch attendance records",
		})
	}

	var responses []dto.AttendanceRecordResponse
	for _, r := range records {
		responses = append(responses, dto.ToAttendanceRecordResponse(&r))
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    responses,
	})
}

// GetCurrentQR godoc
// @Summary Get current active QR token for a lesson
// @Tags attendance
// @Produce json
// @Param id path string true "Lesson ID"
// @Success 200 {object} dto.QRTokenResponse
// @Router /api/v1/lessons/{id}/qr [get]
func (h *AttendanceHandler) GetCurrentQR(c *fiber.Ctx) error {
	lessonID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid lesson ID",
		})
	}

	token, err := h.attendanceService.GetCurrentQRToken(c.Context(), lessonID)
	if err != nil {
		return handleAttendanceError(c, err)
	}
	if token == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "No active QR token",
		})
	}

	lid, _ := uuid.Parse(token.LessonID)
	return c.JSON(fiber.Map{
		"success": true,
		"data": dto.QRTokenResponse{
			LessonID:  lid,
			Payload:   token.Payload,
			Signature: token.Signature,
			IssuedAt:  token.IssuedAt,
			ExpiresAt: token.ExpiresAt,
		},
	})
}

// RotateQR godoc
// @Summary Force rotate QR token for a lesson
// @Tags attendance
// @Produce json
// @Param id path string true "Lesson ID"
// @Success 200 {object} dto.QRTokenResponse
// @Router /api/v1/lessons/{id}/qr/rotate [post]
func (h *AttendanceHandler) RotateQR(c *fiber.Ctx) error {
	lessonID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid lesson ID",
		})
	}

	token, err := h.attendanceService.RotateQRToken(c.Context(), lessonID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to rotate QR token",
		})
	}

	lid, _ := uuid.Parse(token.LessonID)
	return c.JSON(fiber.Map{
		"success": true,
		"data": dto.QRTokenResponse{
			LessonID:  lid,
			Payload:   token.Payload,
			Signature: token.Signature,
			IssuedAt:  token.IssuedAt,
			ExpiresAt: token.ExpiresAt,
		},
		"message": "QR token rotated successfully",
	})
}

// ManualOverride godoc
// @Summary Manually override a student's attendance status
// @Tags attendance
// @Accept json
// @Produce json
// @Param id path string true "Lesson ID"
// @Param body body dto.ManualOverrideRequest true "Override data"
// @Success 200 {object} fiber.Map
// @Router /api/v1/attendance/lesson/{id}/override [post]
func (h *AttendanceHandler) ManualOverride(c *fiber.Ctx) error {
	lessonID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid lesson ID",
		})
	}

	var req dto.ManualOverrideRequest
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

	studentID, _ := uuid.Parse(req.StudentID)

	input := attendanceApp.ManualOverrideInput{
		LessonID:     lessonID,
		StudentID:    studentID,
		OverriddenBy: teacherID,
		Status:       attendanceDomain.AttendanceStatus(req.Status),
		Reason:       req.Reason,
	}

	if err := h.attendanceService.ManualOverride(c.Context(), input); err != nil {
		return handleAttendanceError(c, err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Attendance record overridden successfully",
	})
}

func handleAttendanceError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, attendanceApp.ErrLessonNotFound):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "Lesson not found",
		})
	case errors.Is(err, attendanceApp.ErrLessonNotLive):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Lesson is not currently live",
		})
	case errors.Is(err, attendanceApp.ErrSessionNotActive):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Attendance session is not active",
		})
	case errors.Is(err, attendanceApp.ErrNotEnrolled):
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error":   "You are not enrolled in this course",
		})
	case errors.Is(err, attendanceApp.ErrInvalidQRToken):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid QR code",
		})
	case errors.Is(err, attendanceApp.ErrQRTokenExpired):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "QR code has expired, please scan the new one",
		})
	case errors.Is(err, attendanceApp.ErrQRNonceConsumed):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "This QR code has already been used",
		})
	case errors.Is(err, attendanceApp.ErrOutsideGeofence):
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error":   "You are outside the allowed location",
		})
	case errors.Is(err, attendanceApp.ErrRateLimitExceeded):
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"success": false,
			"error":   "Too many scan attempts, please wait",
		})
	case errors.Is(err, attendanceApp.ErrEmulatorDetected):
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error":   "Emulator detected, please use a physical device",
		})
	case errors.Is(err, attendanceApp.ErrAlreadyScanned):
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"success": false,
			"error":   "Attendance already recorded",
		})
	case errors.Is(err, attendanceApp.ErrSharedDeviceViolation):
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error":   "Shared device detected. You cannot scan for multiple students using the same device.",
		})
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Internal server error",
		})
	}
}

// GetStudentCourseAnalytics godoc
// @Summary Get analytics for a student in a specific course
// @Tags attendance
// @Produce json
// @Param studentId path string true "Student ID"
// @Param courseId path string true "Course ID"
// @Success 200 {object} dto.StudentAnalyticsResponse
// @Router /api/v1/attendance/student/{studentId}/course/{courseId}/analytics [get]
func (h *AttendanceHandler) GetStudentCourseAnalytics(c *fiber.Ctx) error {
	studentID, err := uuid.Parse(c.Params("studentId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid student ID",
		})
	}

	courseID, err := uuid.Parse(c.Params("courseId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid course ID",
		})
	}

	analytics, err := h.attendanceService.GetStudentCourseAnalytics(c.Context(), studentID, courseID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to fetch analytics",
		})
	}

	// Fetch student info
	userInfo, err := h.authClient.GetUserInfo(c.Context(), studentID.String())
	studentName := ""
	studentProfileImg := ""
	if err == nil && userInfo != nil {
		studentName = userInfo.Name
		studentProfileImg = userInfo.ProfileImg
	}

	// Convert to DTO
	weeklyAttendance := make([]dto.DailyAttendance, len(analytics.WeeklyAttendance))
	for i, wa := range analytics.WeeklyAttendance {
		weeklyAttendance[i] = dto.DailyAttendance{
			Day:   wa.Day,
			Hours: wa.Hours,
		}
	}

	recentActivity := make([]dto.RecentActivityItem, len(analytics.RecentActivity))
	for i, ra := range analytics.RecentActivity {
		recentActivity[i] = dto.RecentActivityItem{
			LessonID:     ra.LessonID,
			LessonTitle:  ra.LessonTitle,
			Status:       ra.Status,
			ScheduledAt:  ra.ScheduledAt,
			ScannedAt:    ra.ScannedAt,
			DurationMins: ra.DurationMins,
		}
	}

	response := dto.StudentAnalyticsResponse{
		StudentID:         studentID,
		StudentName:       studentName,
		StudentProfileImg: studentProfileImg,
		CourseID:          courseID,
		AttendanceRate:    analytics.AttendanceRate,
		AttendanceChange:  analytics.AttendanceChange,
		CompletionRate:    analytics.CompletionRate,
		CompletedLessons:  analytics.CompletedLessons,
		TotalLessons:      analytics.TotalLessons,
		WeeklyAttendance:  weeklyAttendance,
		Rank:              analytics.Rank,
		TotalStudents:     analytics.TotalStudents,
		Points:            analytics.Points,
		RecentActivity:    recentActivity,
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    response,
	})
}
