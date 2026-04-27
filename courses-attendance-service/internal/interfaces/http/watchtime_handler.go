package http

import (
	"errors"

	watchtimeApp "github.com/OmarrGhorab/courses-attendance-service/internal/application/watchtime"
	"github.com/OmarrGhorab/courses-attendance-service/internal/domain/watchtime"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/authclient"
	"github.com/OmarrGhorab/courses-attendance-service/internal/interfaces/http/dto"
	"github.com/OmarrGhorab/courses-attendance-service/internal/interfaces/http/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// WatchTimeHandler handles watch time tracking HTTP requests
type WatchTimeHandler struct {
	watchService *watchtimeApp.Service
	authClient   *authclient.Client
}

func NewWatchTimeHandler(watchService *watchtimeApp.Service, authClient *authclient.Client) *WatchTimeHandler {
	return &WatchTimeHandler{
		watchService: watchService,
		authClient:   authClient,
	}
}

func (h *WatchTimeHandler) RegisterRoutes(router fiber.Router) {
	auth := middleware.Authenticate(h.authClient)
	teacherOnly := middleware.RequireRole("TEACHER", "INSTRUCTOR")

	watch := router.Group("/watch", auth)
	watch.Post("/heartbeat", h.RecordHeartbeat)
	watch.Post("/preview/heartbeat", h.RecordPreviewHeartbeat) // NEW: Preview tracking
	watch.Get("/preview/:courseId/progress", h.GetPreviewProgress) // NEW: Get preview progress
	watch.Get("/lesson/:lessonId/progress", h.GetLessonProgress)
	watch.Get("/course/:courseId/progress", h.GetCourseProgress)
	watch.Get("/dashboard", h.GetDashboard)
	watch.Get("/course/:courseId/leaderboard", teacherOnly, h.GetLeaderboard)
	watch.Get("/recommendations/profile", h.GetRecommendationProfile)
	watch.Post("/course/:courseId/recompute", h.RecomputeCourseAnalytics) // Manual trigger for testing
}

// RecordHeartbeat godoc
// @Summary Record a video watch heartbeat
// @Description Called every ~15 seconds by the client while a video is playing.
// @Tags watch
// @Accept json
// @Produce json
// @Param body body dto.WatchHeartbeatRequest true "Heartbeat data"
// @Success 200 {object} dto.LessonProgressResponse
// @Router /api/v1/watch/heartbeat [post]
func (h *WatchTimeHandler) RecordHeartbeat(c *fiber.Ctx) error {
	var req dto.WatchHeartbeatRequest
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

	lessonID, err := uuid.Parse(req.LessonID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid lesson ID",
		})
	}

	deviceType := watchtime.DeviceTypeDesktop
	if req.DeviceType != "" {
		deviceType = watchtime.DeviceType(req.DeviceType)
	}

	input := watchtimeApp.RecordWatchInput{
		LessonID:       lessonID,
		WatchedSeconds: req.WatchedSeconds,
		LastPosition:   req.LastPosition,
		Completed:      req.Completed,
		DeviceType:     deviceType,
	}

	progress, err := h.watchService.RecordWatchEvent(c.Context(), userID, input)
	if err != nil {
		return handleWatchServiceError(c, err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    dto.ToLessonProgressResponse(progress),
	})
}

// GetLessonProgress godoc
// @Summary Get watch progress for a specific lesson
// @Tags watch
// @Produce json
// @Param lessonId path string true "Lesson ID"
// @Success 200 {object} dto.LessonProgressResponse
// @Router /api/v1/watch/lesson/{lessonId}/progress [get]
func (h *WatchTimeHandler) GetLessonProgress(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "Unauthorized",
		})
	}

	lessonID, err := uuid.Parse(c.Params("lessonId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid lesson ID",
		})
	}

	progress, err := h.watchService.GetLessonProgress(c.Context(), userID, lessonID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to get lesson progress",
		})
	}

	if progress == nil {
		return c.JSON(fiber.Map{
			"success": true,
			"data":    nil,
			"message": "No watch data yet for this lesson",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    dto.ToLessonProgressResponse(progress),
	})
}

// GetCourseProgress godoc
// @Summary Get course-level engagement analytics
// @Tags watch
// @Produce json
// @Param courseId path string true "Course ID"
// @Success 200 {object} dto.CourseAnalyticsResponse
// @Router /api/v1/watch/course/{courseId}/progress [get]
func (h *WatchTimeHandler) GetCourseProgress(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "Unauthorized",
		})
	}

	courseID, err := uuid.Parse(c.Params("courseId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid course ID",
		})
	}

	analytics, err := h.watchService.GetCourseAnalytics(c.Context(), userID, courseID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to get course analytics",
		})
	}

	if analytics == nil {
		return c.JSON(fiber.Map{
			"success": true,
			"data":    nil,
			"message": "No watch data yet for this course",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    dto.ToCourseAnalyticsResponse(analytics),
	})
}

// GetDashboard godoc
// @Summary Get student's watch analytics across all enrolled courses
// @Tags watch
// @Produce json
// @Success 200 {object} dto.StudentDashboardResponse
// @Router /api/v1/watch/dashboard [get]
func (h *WatchTimeHandler) GetDashboard(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "Unauthorized",
		})
	}

	dashboard, err := h.watchService.GetStudentDashboard(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to get dashboard",
		})
	}

	// Build response
	totalWatchTime := 0
	totalCompleted := 0
	totalEngagement := 0.0
	courseResponses := make([]dto.CourseAnalyticsResponse, len(dashboard.AllAnalytics))

	for i, a := range dashboard.AllAnalytics {
		totalWatchTime += a.TotalWatchTime
		totalCompleted += a.LessonsCompleted
		totalEngagement += a.EngagementScore
		courseResponses[i] = dto.ToCourseAnalyticsResponse(&dashboard.AllAnalytics[i])
	}

	avgEngagement := 0.0
	if len(dashboard.AllAnalytics) > 0 {
		avgEngagement = totalEngagement / float64(len(dashboard.AllAnalytics))
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": dto.StudentDashboardResponse{
			TotalWatchTime:     totalWatchTime,
			TotalCoursesActive: len(dashboard.AllAnalytics),
			TotalCompleted:     totalCompleted,
			OverallEngagement:  avgEngagement,
			Courses:            courseResponses,
		},
	})
}

// GetLeaderboard godoc
// @Summary Get engagement leaderboard for a course (teacher only)
// @Tags watch
// @Produce json
// @Param courseId path string true "Course ID"
// @Success 200 {array} dto.LeaderboardEntryResponse
// @Router /api/v1/watch/course/{courseId}/leaderboard [get]
func (h *WatchTimeHandler) GetLeaderboard(c *fiber.Ctx) error {
	courseID, err := uuid.Parse(c.Params("courseId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid course ID",
		})
	}

	limit := c.QueryInt("limit", 20)
	entries, err := h.watchService.GetCourseLeaderboard(c.Context(), courseID, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to get leaderboard",
		})
	}

	responses := make([]dto.LeaderboardEntryResponse, len(entries))
	for i, e := range entries {
		responses[i] = dto.LeaderboardEntryResponse{
			Rank:             i + 1,
			UserID:           e.UserID,
			TotalWatchTime:   e.TotalWatchTime,
			LessonsCompleted: e.LessonsCompleted,
			CompletionPct:    e.CompletionPct,
			EngagementScore:  e.EngagementScore,
			LastActivityAt:   e.LastActivityAt,
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    responses,
	})
}

// GetRecommendationProfile godoc
// @Summary Get structured analytics profile for AI course recommendations
// @Tags watch
// @Produce json
// @Success 200 {object} dto.RecommendationProfileResponse
// @Router /api/v1/watch/recommendations/profile [get]
func (h *WatchTimeHandler) GetRecommendationProfile(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "Unauthorized",
		})
	}

	profile, err := h.watchService.GetRecommendationProfile(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to get recommendation profile",
		})
	}

	// Build subject preferences
	subjectPrefs := make([]dto.SubjectPreferenceResponse, len(profile.SubjectPreferences))
	for i, sp := range profile.SubjectPreferences {
		subjectPrefs[i] = dto.SubjectPreferenceResponse{
			SubjectID:        sp.SubjectID,
			SubjectName:      sp.SubjectName,
			TotalWatchTime:   sp.TotalWatchTime,
			CoursesWatched:   sp.CoursesWatched,
			AvgEngagement:    sp.AvgEngagement,
			AvgCompletionPct: sp.AvgCompletionPct,
		}
	}

	// Build preview interests
	previewInterests := make([]dto.PreviewInterestResponse, len(profile.PreviewInterests))
	for i, pi := range profile.PreviewInterests {
		previewInterests[i] = dto.PreviewInterestResponse{
			SubjectID:        pi.SubjectID,
			SubjectName:      pi.SubjectName,
			TotalWatchTime:   pi.TotalWatchTime,
			CoursesViewed:    pi.CoursesViewed,
			AvgCompletionPct: pi.AvgCompletionPct,
		}
	}

	// Build preview progress
	previewProgress := make([]dto.PreviewProgressResponse, len(profile.AllPreviewProgress))
	for i := range profile.AllPreviewProgress {
		previewProgress[i] = dto.ToPreviewProgressResponse(&profile.AllPreviewProgress[i])
	}

	// Build course analytics
	courseAnalytics := make([]dto.CourseAnalyticsResponse, len(profile.AllAnalytics))
	totalWatchTime := 0
	totalCompleted := 0
	totalEngagement := 0.0
	for i, a := range profile.AllAnalytics {
		totalWatchTime += a.TotalWatchTime
		totalCompleted += a.LessonsCompleted
		totalEngagement += a.EngagementScore
		courseAnalytics[i] = dto.ToCourseAnalyticsResponse(&profile.AllAnalytics[i])
	}

	avgEngagement := 0.0
	if len(profile.AllAnalytics) > 0 {
		avgEngagement = totalEngagement / float64(len(profile.AllAnalytics))
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": dto.RecommendationProfileResponse{
			UserID:             userID,
			TotalWatchTime:     totalWatchTime,
			TotalCoursesActive: len(profile.AllAnalytics),
			TotalCompleted:     totalCompleted,
			OverallEngagement:  avgEngagement,
			SubjectPreferences: subjectPrefs,
			PreviewInterests:   previewInterests,
			CourseAnalytics:    courseAnalytics,
			PreviewProgress:    previewProgress,
			WatchPatterns: dto.WatchPatternsResponse{
				AvgSessionDuration:  profile.AvgSessionDuration,
				PreferredDeviceType: profile.PreferredDevice,
				CompletionTendency:  profile.CompletionTendency,
				AvgCompletionPct:    profile.AvgCompletionPct,
			},
		},
	})
}

// RecomputeCourseAnalytics godoc
// @Summary Manually trigger course analytics recomputation (for testing/debugging)
// @Tags watch
// @Produce json
// @Param courseId path string true "Course ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/watch/course/{courseId}/recompute [post]
func (h *WatchTimeHandler) RecomputeCourseAnalytics(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "Unauthorized",
		})
	}

	courseID, err := uuid.Parse(c.Params("courseId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid course ID",
		})
	}

	// Trigger manual recompute
	err = h.watchService.ManualRecomputeCourseAnalytics(c.Context(), courseID, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Course analytics recomputed successfully",
	})
}

// RecordPreviewHeartbeat godoc
// @Summary Record a preview video watch heartbeat (for non-enrolled users)
// @Description Called every ~3-5 seconds by the client while a preview video is playing.
// @Tags watch
// @Accept json
// @Produce json
// @Param body body dto.PreviewHeartbeatRequest true "Preview heartbeat data"
// @Success 200 {object} dto.PreviewProgressResponse
// @Router /api/v1/watch/preview/heartbeat [post]
func (h *WatchTimeHandler) RecordPreviewHeartbeat(c *fiber.Ctx) error {
	var req dto.PreviewHeartbeatRequest
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

	courseID, err := uuid.Parse(req.CourseID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid course ID",
		})
	}

	deviceType := watchtime.DeviceTypeDesktop
	if req.DeviceType != "" {
		deviceType = watchtime.DeviceType(req.DeviceType)
	}

	input := watchtimeApp.RecordPreviewInput{
		CourseID:       courseID,
		WatchedSeconds: req.WatchedSeconds,
		LastPosition:   req.LastPosition,
		Completed:      req.Completed,
		DeviceType:     deviceType,
	}

	progress, err := h.watchService.RecordPreviewWatchEvent(c.Context(), userID, input)
	if err != nil {
		if err.Error() == "user is already enrolled, use regular watch heartbeat instead" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"error":   err.Error(),
			})
		}
		return handleWatchServiceError(c, err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    dto.ToPreviewProgressResponse(progress),
	})
}

// GetPreviewProgress godoc
// @Summary Get watch progress for a course preview
// @Tags watch
// @Produce json
// @Param courseId path string true "Course ID"
// @Success 200 {object} dto.PreviewProgressResponse
// @Router /api/v1/watch/preview/{courseId}/progress [get]
func (h *WatchTimeHandler) GetPreviewProgress(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "Unauthorized",
		})
	}

	courseID, err := uuid.Parse(c.Params("courseId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid course ID",
		})
	}

	progress, err := h.watchService.GetPreviewProgress(c.Context(), userID, courseID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to get preview progress",
		})
	}

	if progress == nil {
		return c.JSON(fiber.Map{
			"success": true,
			"data":    nil,
			"message": "No preview watch data yet for this course",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    dto.ToPreviewProgressResponse(progress),
	})
}

func handleWatchServiceError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, watchtimeApp.ErrLessonNotFound):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "Lesson not found",
		})
	case errors.Is(err, watchtimeApp.ErrNotEnrolled):
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error":   "You are not enrolled in this course",
		})
	case errors.Is(err, watchtimeApp.ErrNotOnlineLesson):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Watch tracking is only available for online lessons",
		})
	case errors.Is(err, watchtimeApp.ErrInvalidInput):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid input data",
		})
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Internal server error",
		})
	}
}
