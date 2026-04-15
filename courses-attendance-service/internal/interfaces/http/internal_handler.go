package http

import (
	"github.com/OmarrGhorab/courses-attendance-service/internal/application/course"
	watchtimeApp "github.com/OmarrGhorab/courses-attendance-service/internal/application/watchtime"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/authclient"
	"github.com/OmarrGhorab/courses-attendance-service/internal/interfaces/http/dto"
	"github.com/OmarrGhorab/courses-attendance-service/internal/interfaces/http/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type InternalHandler struct {
	courseService  *course.Service
	watchService   *watchtimeApp.Service
	authClient     *authclient.Client
	internalSecret string
}

func NewInternalHandler(courseService *course.Service, watchService *watchtimeApp.Service, authClient *authclient.Client, internalSecret string) *InternalHandler {
	return &InternalHandler{
		courseService:  courseService,
		watchService:   watchService,
		authClient:     authClient,
		internalSecret: internalSecret,
	}
}

func (h *InternalHandler) RegisterRoutes(router fiber.Router) {
	internal := router.Group("/internal", middleware.InternalOnly(h.internalSecret))

	internal.Get("/courses/:id", h.GetCourse)
	internal.Post("/enrollments/activate", h.ActivateEnrollment)
	internal.Get("/enrollments/check", h.CheckEnrollment)
	internal.Post("/enrollments", h.InternalEnroll)
	internal.Get("/courses", h.ListAllCourses)
	internal.Get("/analytics/user/:userId", h.GetUserAnalytics)
	internal.Get("/reports/student/:userId/weekly", h.GetWeeklyReport)
}

func (h *InternalHandler) InternalEnroll(c *fiber.Ctx) error {
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

	userID, _ := uuid.Parse(req.UserID)
	courseID, _ := uuid.Parse(req.CourseID)

	enrollment, err := h.courseService.EnrollStudent(c.Context(), courseID, userID)
	if err != nil && err.Error() != "student already enrolled in this course" {
		return handleServiceError(c, err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    dto.ToEnrollmentResponse(enrollment),
	})
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

	resp := dto.ToCourseResponse(course)

	// Populate teacher info
	if h.authClient != nil {
		tInfo, _ := h.authClient.GetUserInfo(c.Context(), course.TeacherID.String())
		if tInfo != nil {
			resp.TeacherName = tInfo.Name
			resp.TeacherProfileImg = tInfo.ProfileImg
		}
	}

	// Populate teacher authority
	authority, _ := h.courseService.GetTeacherAuthority(c.Context(), course.TeacherID)
	resp.TeacherAuthority = authority

	return c.JSON(fiber.Map{
		"success": true,
		"data":    resp,
	})
}

func (h *InternalHandler) ActivateEnrollment(c *fiber.Ctx) error {
	var req struct {
		UserID    string `json:"userId"`
		CourseID  string `json:"courseId"`
		PeriodKey string `json:"periodKey"`
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

	err = h.courseService.MarkEnrollmentPaid(c.Context(), courseID, userID, req.PeriodKey)
	if err != nil {
		return handleServiceError(c, err)
	}


	return c.JSON(fiber.Map{
		"success": true,
		"message": "Enrollment activated successfully",
	})
}

func (h *InternalHandler) CheckEnrollment(c *fiber.Ctx) error {
	userIDStr := c.Query("userId")
	courseIDStr := c.Query("courseId")

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid user ID",
		})
	}

	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid course ID",
		})
	}

	isEnrolled, err := h.courseService.IsEnrolled(c.Context(), courseID, userID)
	if err != nil {
		return handleServiceError(c, err)
	}

	var isPaid bool
	if isEnrolled {
		enrollment, _ := h.courseService.GetEnrollment(c.Context(), courseID, userID)
		if enrollment != nil {
			isPaid = enrollment.IsPaid
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"isEnrolled": isEnrolled,
			"isPaid":     isPaid,
		},
	})
}

func (h *InternalHandler) ListAllCourses(c *fiber.Ctx) error {
	courses, err := h.courseService.ListCourses(c.Context(), nil)
	if err != nil {
		return handleServiceError(c, err)
	}

	teacherAuthorityMap := make(map[uuid.UUID]int)
	teacherInfoMap := make(map[uuid.UUID]*authclient.UserInfo)
	responses := make([]dto.CourseResponse, len(courses))

	for i, crs := range courses {
		authority, ok := teacherAuthorityMap[crs.TeacherID]
		if !ok {
			authority, _ = h.courseService.GetTeacherAuthority(c.Context(), crs.TeacherID)
			teacherAuthorityMap[crs.TeacherID] = authority
		}
		
		tInfo, ok := teacherInfoMap[crs.TeacherID]
		if !ok {
			tInfo, _ = h.authClient.GetUserInfo(c.Context(), crs.TeacherID.String())
			teacherInfoMap[crs.TeacherID] = tInfo
		}
		
		resp := dto.ToCourseResponse(&crs)
		resp.TeacherAuthority = authority
		
		if tInfo != nil {
			resp.TeacherName = tInfo.Name
			resp.TeacherProfileImg = tInfo.ProfileImg
		}

		responses[i] = resp
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    responses,
	})
}

func (h *InternalHandler) GetUserAnalytics(c *fiber.Ctx) error {
	userId, err := uuid.Parse(c.Params("userId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid user ID",
		})
	}

	profile, err := h.watchService.GetRecommendationProfile(c.Context(), userId)
	if err != nil {
		return handleServiceError(c, err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    profile,
	})
}

func (h *InternalHandler) GetWeeklyReport(c *fiber.Ctx) error {
	userId, err := uuid.Parse(c.Params("userId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid user ID",
		})
	}

	reportData, err := h.watchService.GetWeeklyReportData(c.Context(), userId)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    reportData,
	})
}


