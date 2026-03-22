package http

import (
	"errors"
	"fmt"
	"time"


	courseApp "github.com/OmarrGhorab/courses-attendance-service/internal/application/course"
	lessonApp "github.com/OmarrGhorab/courses-attendance-service/internal/application/lesson"
	progressApp "github.com/OmarrGhorab/courses-attendance-service/internal/application/progress"
	courseDomain "github.com/OmarrGhorab/courses-attendance-service/internal/domain/course"
	lessonDomain "github.com/OmarrGhorab/courses-attendance-service/internal/domain/lesson"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/authclient"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/persistence/postgres"
	"github.com/OmarrGhorab/courses-attendance-service/internal/interfaces/http/dto"
	"github.com/OmarrGhorab/courses-attendance-service/internal/interfaces/http/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// CourseHandler handles course-related HTTP requests
type CourseHandler struct {
	courseService    *courseApp.Service
	lessonService    *lessonApp.Service
	progressService  *progressApp.Service
	authClient       *authclient.Client
	ratingRepo       *postgres.TeacherRatingRepository
	courseRatingRepo *postgres.CourseRatingRepository
	enrollmentRepo   *postgres.EnrollmentRepository
	attendanceRepo   *postgres.AttendanceRecordRepository
	absenceRepo      *postgres.AbsenceRequestRepository
}

func NewCourseHandler(courseService *courseApp.Service, authClient *authclient.Client) *CourseHandler {
	return &CourseHandler{
		courseService: courseService,
		authClient:    authClient,
	}
}

// NewCourseHandlerWithServices creates a handler with all required services for combined endpoints
func NewCourseHandlerWithServices(
	courseService *courseApp.Service,
	lessonService *lessonApp.Service,
	progressService *progressApp.Service,
	authClient *authclient.Client,
	ratingRepo *postgres.TeacherRatingRepository,
	courseRatingRepo *postgres.CourseRatingRepository,
	enrollmentRepo *postgres.EnrollmentRepository,
	attendanceRepo *postgres.AttendanceRecordRepository,
	absenceRepo *postgres.AbsenceRequestRepository,
) *CourseHandler {
	return &CourseHandler{
		courseService:    courseService,
		lessonService:    lessonService,
		progressService:  progressService,
		authClient:       authClient,
		ratingRepo:       ratingRepo,
		courseRatingRepo: courseRatingRepo,
		enrollmentRepo:   enrollmentRepo,
		attendanceRepo:   attendanceRepo,
		absenceRepo:      absenceRepo,
	}
}

func (h *CourseHandler) RegisterRoutes(router fiber.Router) {
	// Standard auth middleware for all course routes
	auth := middleware.Authenticate(h.authClient)

	courses := router.Group("/courses", auth)

	// Teacher/Instructor only routes
	teacherOnly := middleware.RequireRole("TEACHER", "INSTRUCTOR")
	courses.Post("/", teacherOnly, h.CreateCourse)
	courses.Patch("/:id", teacherOnly, h.UpdateCourse)
	courses.Post("/:id/assistants", teacherOnly, h.AddAssistant)
	courses.Get("/:id/assistants", teacherOnly, h.GetCourseAssistants)
	courses.Delete("/:id/assistants/:assistantId", teacherOnly, h.RemoveAssistant)
	courses.Get("/:id/enrollments", teacherOnly, h.GetCourseEnrollments)

	// Public/Shared routes (but still authenticated)
	courses.Get("/", h.ListCourses)
	courses.Get("/my", h.GetMyCourses)
	courses.Get("/my-subjects", h.GetMySubjects)
	courses.Get("/subjects/:subjectId/details", h.GetSubjectDetails) // NEW: Subject with courses
	courses.Get("/:id/details", h.GetCourseDetails) // Combined endpoint
	courses.Get("/:id/reviews", h.GetCourseReviews) // NEW: Course reviews
	courses.Post("/:id/reviews", h.CreateCourseReview) // NEW: Create review
	courses.Put("/:id/reviews/:reviewId", h.UpdateCourseReview) // NEW: Update review
	courses.Delete("/:id/reviews/:reviewId", h.DeleteCourseReview) // NEW: Delete review
	courses.Get("/:id", h.GetCourse)
	courses.Post("/:id/enroll", h.EnrollStudent)

	// Subject routes
	router.Get("/subjects", auth, h.GetSubjects)
	router.Post("/subjects", auth, teacherOnly, h.CreateSubject)
}


// CreateCourse godoc
// @Summary Create a new course
// @Tags courses
// @Accept json
// @Produce json
// @Param body body dto.CreateCourseRequest true "Course data"
// @Success 201 {object} dto.CourseResponse
// @Router /api/v1/courses [post]
func (h *CourseHandler) CreateCourse(c *fiber.Ctx) error {
	var req dto.CreateCourseRequest
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

	// Get teacher ID from auth context (middleware will set this)
	teacherID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "Unauthorized",
		})
	}

	subjectID, _ := uuid.Parse(req.SubjectID)

	input := courseApp.CreateCourseInput{
		Title:                   req.Title,
		Description:             req.Description,
		SubjectID:               subjectID,
		TeacherID:               teacherID,
		CourseImage:             req.CourseImage,
		DeliveryType:            courseDomain.DeliveryType(req.DeliveryType),
		LocationName:            req.LocationName,
		LocationLat:             req.LocationLat,
		LocationLng:             req.LocationLng,
		GeofenceRadiusM:         req.GeofenceRadiusM,
		TotalLessons:            req.TotalLessons,
		AttendanceWindowMinutes: req.AttendanceWindowMinutes,
		Price:                   req.Price,
		Currency:                req.Currency,
		IsPaid:                  req.IsPaid,
		AttendanceWeight:        req.AttendanceWeight,
	}

	course, err := h.courseService.CreateCourse(c.Context(), input)
	if err != nil {
		return handleServiceError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    dto.ToCourseResponse(course),
	})
}

// ListCourses godoc
// @Summary List all courses or filter by subject
// @Tags courses
// @Produce json
// @Param subjectId query string false "Subject ID"
// @Param deliveryType query string false "Delivery type (ONLINE, OFFLINE)"
// @Param isPaid query string false "Is paid (true, false)"
// @Param billingType query string false "Billing type (ONE_TIME, MONTHLY)"
// @Param status query string false "Status (ACTIVE, PAUSED, ARCHIVED)"
// @Param minPrice query number false "Minimum price"
// @Param maxPrice query number false "Maximum price"
// @Success 200 {array} dto.CourseListResponse
// @Router /api/v1/courses [get]
func (h *CourseHandler) ListCourses(c *fiber.Ctx) error {
	// Parse filters
	var subjectIDPtr *uuid.UUID
	subjectIDStr := c.Query("subjectId")
	if subjectIDStr != "" {
		id, err := uuid.Parse(subjectIDStr)
		if err == nil {
			subjectIDPtr = &id
		}
	}

	deliveryType := c.Query("deliveryType")
	isPaidStr := c.Query("isPaid")
	billingType := c.Query("billingType")
	statusFilter := c.Query("status")
	minPriceStr := c.Query("minPrice")
	maxPriceStr := c.Query("maxPrice")

	// Get courses
	courses, err := h.courseService.ListCourses(c.Context(), subjectIDPtr)
	if err != nil {
		return handleServiceError(c, err)
	}

	// Apply filters
	var filteredCourses []courseDomain.Course
	for _, crs := range courses {
		// Delivery type filter
		if deliveryType != "" && string(crs.DeliveryType) != deliveryType {
			continue
		}

		// Is paid filter
		if isPaidStr != "" {
			isPaid := isPaidStr == "true"
			if crs.IsPaid != isPaid {
				continue
			}
		}

		// Billing type filter
		if billingType != "" && string(crs.BillingType) != billingType {
			continue
		}

		// Status filter
		if statusFilter != "" && string(crs.Status) != statusFilter {
			continue
		}

		// Price range filter
		if minPriceStr != "" {
			// Parse minPrice (simple implementation)
			if crs.Price < 0 { // You can add proper parsing here
				continue
			}
		}
		if maxPriceStr != "" {
			// Parse maxPrice (simple implementation)
			if crs.Price > 999999 { // You can add proper parsing here
				continue
			}
		}

		filteredCourses = append(filteredCourses, crs)
	}

	// Build enhanced responses with teacher info
	var responses []dto.CourseListResponse
	
	// Collect all teacher IDs and course IDs for batch lookups
	teacherIDs := make([]uuid.UUID, 0, len(filteredCourses))
	courseIDs := make([]uuid.UUID, 0, len(filteredCourses))
	teacherIDMap := make(map[uuid.UUID]bool)
	
	for _, crs := range filteredCourses {
		if !teacherIDMap[crs.TeacherID] {
			teacherIDs = append(teacherIDs, crs.TeacherID)
			teacherIDMap[crs.TeacherID] = true
		}
		courseIDs = append(courseIDs, crs.ID)
	}

	// Batch fetch teacher ratings
	var teacherRatingsMap map[uuid.UUID]float64
	if h.ratingRepo != nil && len(teacherIDs) > 0 {
		teacherRatingsMap, _ = h.ratingRepo.GetMultipleTeacherAvgRatings(c.Context(), teacherIDs)
	}

	// Batch fetch course ratings
	var courseRatingsMap map[uuid.UUID]*postgres.CourseAvgRating
	if h.courseRatingRepo != nil && len(courseIDs) > 0 {
		courseRatingsMap, _ = h.courseRatingRepo.GetMultipleCourseAvgRatings(c.Context(), courseIDs)
	}

	// Batch fetch enrollment counts
	enrollmentCounts := make(map[uuid.UUID]int)
	if h.enrollmentRepo != nil {
		for _, courseID := range courseIDs {
			enrollments, err := h.enrollmentRepo.GetByCourseID(c.Context(), courseID)
			if err == nil {
				enrollmentCounts[courseID] = len(enrollments)
			}
		}
	}

	for _, crs := range filteredCourses {
		response := dto.CourseListResponse{
			ID:                      crs.ID,
			Title:                   crs.Title,
			Description:             crs.Description,
			SubjectID:               crs.SubjectID,
			TeacherID:               crs.TeacherID,
			CourseImage:             crs.CourseImage,
			DeliveryType:            string(crs.DeliveryType),
			LocationName:            crs.LocationName,
			LocationLat:             crs.LocationLat,
			LocationLng:             crs.LocationLng,
			GeofenceRadiusM:         crs.GeofenceRadiusM,
			TotalLessons:            crs.TotalLessons,
			AttendanceWindowMinutes: crs.AttendanceWindowMinutes,
			Price:                   crs.Price,
			Currency:                crs.Currency,
			IsPaid:                  crs.IsPaid,
			BillingType:             string(crs.BillingType),
			Status:                  string(crs.Status),
			AttendanceWeight:        crs.AttendanceWeight,
			CreatedAt:               crs.CreatedAt,
			UpdatedAt:               crs.UpdatedAt,
		}

		// Add subject name
		if crs.Subject != nil {
			response.SubjectName = crs.Subject.Name
		}

		// Fetch teacher info
		if h.authClient != nil {
			userInfo, err := h.authClient.GetUserInfo(c.Context(), crs.TeacherID.String())
			if err == nil && userInfo != nil {
				response.TeacherName = userInfo.Name
				response.TeacherProfileImg = userInfo.ProfileImg
			}
		}

		// Add teacher rating from batch lookup
		if teacherRatingsMap != nil {
			if rating, ok := teacherRatingsMap[crs.TeacherID]; ok {
				response.TeacherRating = rating
			}
		}

		// Add course rating from batch lookup
		if courseRatingsMap != nil {
			if rating, ok := courseRatingsMap[crs.ID]; ok {
				response.CourseRating = rating.AvgRating
				response.TotalRatings = rating.TotalRatings
			}
		}

		// Add enrollment count
		if count, ok := enrollmentCounts[crs.ID]; ok {
			response.EnrolledStudents = count
		}

		responses = append(responses, response)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    responses,
		"total":   len(responses),
		"filters": fiber.Map{
			"subjectId":    subjectIDStr,
			"deliveryType": deliveryType,
			"isPaid":       isPaidStr,
			"billingType":  billingType,
			"status":       statusFilter,
		},
	})
}

// GetCourse godoc
// @Summary Get a course by ID
// @Tags courses
// @Produce json
// @Param id path string true "Course ID"
// @Success 200 {object} dto.CourseResponse
// @Router /api/v1/courses/{id} [get]
func (h *CourseHandler) GetCourse(c *fiber.Ctx) error {
	courseID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid course ID",
		})
	}

	course, err := h.courseService.GetCourse(c.Context(), courseID)
	if err != nil {
		return handleServiceError(c, err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    dto.ToCourseResponse(course),
	})
}

// GetMyCourses godoc
// @Summary Get courses the current user is enrolled in
// @Tags courses
// @Produce json
// @Success 200 {array} dto.CourseResponse
// @Router /api/v1/courses/my [get]
func (h *CourseHandler) GetMyCourses(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "Unauthorized",
		})
	}

	role := c.Locals("userRole").(string)

	var courses []courseDomain.Course
	if role == "TEACHER" || role == "INSTRUCTOR" {
		courses, err = h.courseService.GetTeacherCourses(c.Context(), userID)
	} else {
		courses, err = h.courseService.GetStudentCourses(c.Context(), userID)
	}

	if err != nil {
		return handleServiceError(c, err)
	}

	var responses []dto.CourseResponse
	for _, crs := range courses {
		responses = append(responses, dto.ToCourseResponse(&crs))
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    responses,
	})
}

// GetMySubjects godoc
// @Summary Get subjects the current user is enrolled in
// @Tags subjects
// @Produce json
// @Success 200 {array} dto.SubjectResponse
// @Router /api/v1/courses/my-subjects [get]
func (h *CourseHandler) GetMySubjects(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "Unauthorized",
		})
	}

	subjects, err := h.courseService.GetStudentSubjects(c.Context(), userID)
	if err != nil {
		return handleServiceError(c, err)
	}

	var responses []dto.SubjectResponse
	for _, s := range subjects {
		responses = append(responses, dto.ToSubjectResponse(&s))
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    responses,
	})
}

// UpdateCourse godoc
// @Summary Update a course
// @Tags courses
// @Accept json
// @Produce json
// @Param id path string true "Course ID"
// @Param body body dto.UpdateCourseRequest true "Update data"
// @Success 200 {object} dto.CourseResponse
// @Router /api/v1/courses/{id} [patch]
func (h *CourseHandler) UpdateCourse(c *fiber.Ctx) error {
	courseID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid course ID",
		})
	}

	var req dto.UpdateCourseRequest
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

	input := courseApp.UpdateCourseInput{
		Title:                   req.Title,
		Description:             req.Description,
		LocationName:            req.LocationName,
		LocationLat:             req.LocationLat,
		LocationLng:             req.LocationLng,
		GeofenceRadiusM:         req.GeofenceRadiusM,
		AttendanceWindowMinutes: req.AttendanceWindowMinutes,
		Price:                   req.Price,
	}
	if req.Status != nil {
		status := courseDomain.CourseStatus(*req.Status)
		input.Status = &status
	}

	course, err := h.courseService.UpdateCourse(c.Context(), courseID, teacherID, input)
	if err != nil {
		return handleServiceError(c, err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    dto.ToCourseResponse(course),
	})
}

// EnrollStudent godoc
// @Summary Enroll a student in a course
// @Tags courses
// @Accept json
// @Produce json
// @Param id path string true "Course ID"
// @Param body body dto.EnrollRequest true "Enrollment data"
// @Success 201 {object} dto.EnrollmentResponse
// @Router /api/v1/courses/{id}/enroll [post]
func (h *CourseHandler) EnrollStudent(c *fiber.Ctx) error {
	courseID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid course ID",
		})
	}

	var req dto.EnrollRequest
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

	studentID, _ := uuid.Parse(req.StudentID)

	enrollment, err := h.courseService.EnrollStudent(c.Context(), courseID, studentID)
	if err != nil {
		return handleServiceError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    dto.ToEnrollmentResponse(enrollment),
	})
}

// AddAssistant godoc
// @Summary Add an assistant to a course
// @Tags courses
// @Accept json
// @Produce json
// @Param id path string true "Course ID"
// @Param body body dto.AddAssistantRequest true "Assistant data"
// @Success 201 {object} dto.AssistantResponse
// @Router /api/v1/courses/{id}/assistants [post]
func (h *CourseHandler) AddAssistant(c *fiber.Ctx) error {
	courseID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid course ID",
		})
	}

	var req dto.AddAssistantRequest
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

	assistantID, _ := uuid.Parse(req.AssistantID)

	assistant, err := h.courseService.AddAssistant(c.Context(), courseID, teacherID, assistantID)
	if err != nil {
		return handleServiceError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    dto.ToAssistantResponse(assistant),
	})
}

// GetCourseEnrollments godoc
// @Summary Get all enrollments for a course
// @Tags courses
// @Produce json
// @Param id path string true "Course ID"
// @Success 200 {array} dto.EnrollmentResponse
// @Router /api/v1/courses/{id}/enrollments [get]
func (h *CourseHandler) GetCourseEnrollments(c *fiber.Ctx) error {
	courseID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid course ID",
		})
	}

	enrollments, err := h.courseService.GetCourseEnrollments(c.Context(), courseID)
	if err != nil {
		return handleServiceError(c, err)
	}

	var responses []dto.EnrollmentResponse
	for _, e := range enrollments {
		responses = append(responses, dto.ToEnrollmentResponse(&e))
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    responses,
	})
}

// GetSubjects godoc
// @Summary Get all subjects
// @Tags subjects
// @Produce json
// @Success 200 {array} dto.SubjectResponse
// @Router /api/v1/subjects [get]
func (h *CourseHandler) GetSubjects(c *fiber.Ctx) error {
	subjects, err := h.courseService.ListSubjects(c.Context())
	if err != nil {
		return handleServiceError(c, err)
	}

	var responses []dto.SubjectResponse
	for _, s := range subjects {
		responses = append(responses, dto.ToSubjectResponse(&s))
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    responses,
	})
}

func (h *CourseHandler) CreateSubject(c *fiber.Ctx) error {
	var req struct {
		Name        string `json:"name" validate:"required"`
		Description string `json:"description"`
		Icon        string `json:"icon"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	subject, err := h.courseService.CreateSubject(c.Context(), req.Name, req.Description, req.Icon)
	if err != nil {
		return handleServiceError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    dto.ToSubjectResponse(subject),
	})
}


// Helper functions

func getUserIDFromContext(c *fiber.Ctx) (uuid.UUID, error) {
	userIDStr := c.Locals("userId")
	if userIDStr == nil {
		return uuid.Nil, errors.New("user ID not found in context")
	}

	switch v := userIDStr.(type) {
	case string:
		return uuid.Parse(v)
	case uuid.UUID:
		return v, nil
	default:
		return uuid.Nil, errors.New("invalid user ID type")
	}
}

func handleServiceError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, courseApp.ErrCourseNotFound):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "Course not found",
		})
	case errors.Is(err, courseApp.ErrSubjectNotFound):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "Subject not found",
		})
	case errors.Is(err, courseApp.ErrUnauthorized):
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error":   "You are not authorized to perform this action",
		})
	case errors.Is(err, courseApp.ErrAlreadyEnrolled):
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"success": false,
			"error":   "Student is already enrolled in this course",
		})
	case errors.Is(err, courseApp.ErrAssistantExists):
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"success": false,
			"error":   "Assistant is already added to this course",
		})
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Internal server error",
		})
	}
}


// GetCourseDetails godoc
// @Summary Get complete course details with progress and lessons
// @Tags courses
// @Produce json
// @Param id path string true "Course ID"
// @Param studentId query string false "Student ID (optional, defaults to current user)"
// @Success 200 {object} dto.CourseDetailsResponse
// @Router /api/v1/courses/{id}/details [get]
func (h *CourseHandler) GetCourseDetails(c *fiber.Ctx) error {
	courseID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid course ID",
		})
	}

	// Get current user ID
	currentUserID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "Unauthorized",
		})
	}

	// Get student ID from query or use current user
	studentIDStr := c.Query("studentId")
	var studentID uuid.UUID
	if studentIDStr != "" {
		studentID, err = uuid.Parse(studentIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"error":   "Invalid student ID",
			})
		}
	} else {
		studentID = currentUserID
	}

	// 1. Get course info
	course, err := h.courseService.GetCourse(c.Context(), courseID)
	if err != nil {
		return handleServiceError(c, err)
	}

	// Build course info
	courseInfo := dto.CourseInfo{
		ID:                      course.ID,
		Title:                   course.Title,
		Description:             course.Description,
		SubjectID:               course.SubjectID,
		CourseImage:             course.CourseImage,
		DeliveryType:            string(course.DeliveryType),
		LocationName:            course.LocationName,
		TotalLessons:            course.TotalLessons,
		AttendanceWindowMinutes: course.AttendanceWindowMinutes,
		Price:                   course.Price,
		Currency:                course.Currency,
		IsPaid:                  course.IsPaid,
		BillingType:             string(course.BillingType),
		Status:                  string(course.Status),
		AttendanceWeight:        course.AttendanceWeight,
	}
	if course.Subject != nil {
		courseInfo.SubjectName = course.Subject.Name
	}

	// Get enrollment count
	if h.enrollmentRepo != nil {
		enrollments, err := h.enrollmentRepo.GetByCourseID(c.Context(), courseID)
		if err == nil {
			courseInfo.EnrollmentCount = len(enrollments)
		}
	}

	// Get assistants
	if h.courseService != nil {
		assistants, err := h.courseService.GetCourseAssistants(c.Context(), courseID)
		if err == nil && len(assistants) > 0 {
			var assistantInfos []dto.AssistantInfo
			for _, asst := range assistants {
				assistantInfo := dto.AssistantInfo{
					ID:                asst.ID,
					AssistantID:       asst.AssistantID,
					CanStartLesson:    asst.CanStartLesson,
					CanEndLesson:      asst.CanEndLesson,
					CanViewAttendance: asst.CanViewAttendance,
					CanEditAttendance: asst.CanEditAttendance,
					AddedAt:           asst.CreatedAt,
				}

				// Fetch assistant user info
				if h.authClient != nil {
					userInfo, err := h.authClient.GetUserInfo(c.Context(), asst.AssistantID.String())
					if err == nil && userInfo != nil {
						assistantInfo.AssistantName = userInfo.Name
						assistantInfo.AssistantProfileImg = userInfo.ProfileImg
					}
				}

				assistantInfos = append(assistantInfos, assistantInfo)
			}
			courseInfo.Assistants = assistantInfos
		}
	}

	// 2. Get lessons (if lesson service is available)
	var lessons []dto.LessonInfo
	if h.lessonService != nil {
		lessonList, err := h.lessonService.GetCourseLessons(c.Context(), courseID)
		if err == nil {
			// Collect lesson IDs for batch attendance lookup
			lessonIDs := make([]uuid.UUID, 0, len(lessonList))
			for _, l := range lessonList {
				lessonIDs = append(lessonIDs, l.ID)
			}

			// Batch fetch attendance records for this student
			attendanceMap := make(map[uuid.UUID]string)
			attendeeCountMap := make(map[uuid.UUID]int)
			absenceRequestMap := make(map[uuid.UUID]string)
			
			if h.attendanceRepo != nil && len(lessonIDs) > 0 {
				// Get student's attendance records
				studentRecords, _ := h.attendanceRepo.GetByStudentAndLessons(c.Context(), studentID, lessonIDs)
				for _, record := range studentRecords {
					attendanceMap[record.LessonID] = string(record.Status)
				}

				// Get attendee counts for each lesson
				for _, lessonID := range lessonIDs {
					records, _ := h.attendanceRepo.GetByLessonID(c.Context(), lessonID)
					count := 0
					for _, r := range records {
						if r.Status != "ABSENT" {
							count++
						}
					}
					attendeeCountMap[lessonID] = count
				}
			}

			// Batch fetch absence requests for this student
			if h.absenceRepo != nil && len(lessonIDs) > 0 {
				for _, lessonID := range lessonIDs {
					absenceRequest, _ := h.absenceRepo.GetByLessonAndStudent(c.Context(), lessonID, studentID)
					if absenceRequest != nil {
						absenceRequestMap[lessonID] = string(absenceRequest.Status)
					}
				}
			}

			for _, l := range lessonList {
				lessonInfo := dto.LessonInfo{
					ID:                l.ID,
					Title:             l.Title,
					Description:       l.Description,
					LessonNumber:      l.LessonNumber,
					Status:            string(l.Status),
					ScheduledAt:       l.ScheduledAt,
					StartsAt:          l.StartsAt,
					EndsAt:            l.EndsAt,
					DurationMinutes:   l.DurationMinutes,
					DeliveryType:      string(l.DeliveryType),
					IsFree:            l.IsFree,
					VideoURL:          l.VideoURL,
					VideoPublicID:     l.VideoPublicID,
					MaterialsURL:      l.MaterialsURL,
					Duration:          l.Duration,
					LocationName:      l.LocationName,
					LocationLat:       l.LocationLat,
					LocationLng:       l.LocationLng,
					CanMarkAttendance: l.Status == lessonDomain.LessonStatusLive,
				}

				// Add attendance status if available
				if status, ok := attendanceMap[l.ID]; ok {
					lessonInfo.AttendanceStatus = &status
				}

				// Add attendee count if available
				if count, ok := attendeeCountMap[l.ID]; ok {
					lessonInfo.AttendeeCount = &count
				}

				// Add absence request status if available
				if absenceStatus, ok := absenceRequestMap[l.ID]; ok {
					lessonInfo.AbsenceRequestStatus = &absenceStatus
				}

				lessons = append(lessons, lessonInfo)
			}
		}
	}

	// 3. Get progress (if progress service is available and student ID provided)
	var progressInfo *dto.ProgressInfo
	if h.progressService != nil {
		snapshot, err := h.progressService.GetStudentProgress(c.Context(), courseID, studentID)
		if err == nil && snapshot != nil {
			// Calculate attendance percentage
			totalAttended := snapshot.PresentCount + snapshot.LateCount
			attendancePercentage := 0.0
			if snapshot.TotalLessons > 0 {
				attendancePercentage = (float64(totalAttended) / float64(snapshot.TotalLessons)) * 100
			}

			// Determine status
			status := "Good Standing"
			targetPercentage := 80.0
			if attendancePercentage < 60 {
				status = "At Risk"
			} else if attendancePercentage < targetPercentage {
				status = "Needs Improvement"
			}

			progressInfo = &dto.ProgressInfo{
				AttendancePercentage: attendancePercentage,
				ClassesAttended:      totalAttended,
				TotalClasses:         snapshot.TotalLessons,
				OverallGrade:         snapshot.OverallProgress,
				Status:               status,
				TargetPercentage:     targetPercentage,
				PresentCount:         snapshot.PresentCount,
				LateCount:            snapshot.LateCount,
				AbsentCount:          snapshot.AbsentCount,
				ExcusedCount:         snapshot.ExcusedCount,
				LastUpdated:          snapshot.CalculatedAt,
			}
		}
	}

	// 4. Get teacher info from auth service
	var teacherInfo *dto.TeacherInfo
	if h.authClient != nil {
		userInfo, err := h.authClient.GetUserInfo(c.Context(), course.TeacherID.String())
		if err == nil && userInfo != nil {
			teacherInfo = &dto.TeacherInfo{
				ID:         course.TeacherID,
				Name:       userInfo.Name,
				ProfileImg: userInfo.ProfileImg,
				// Title and Department can be added if available in user metadata
			}
		} else {
			// Fallback if auth service call fails
			teacherInfo = &dto.TeacherInfo{
				ID: course.TeacherID,
			}
		}
	}

	// Build response
	response := dto.CourseDetailsResponse{
		Course:   courseInfo,
		Progress: progressInfo,
		Teacher:  teacherInfo,
		Lessons:  lessons,
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    response,
	})
}


// GetSubjectDetails godoc
// @Summary Get subject with all courses in it
// @Tags subjects
// @Produce json
// @Param subjectId path string true "Subject ID"
// @Success 200 {object} dto.SubjectDetailsResponse
// @Router /api/v1/courses/subjects/{subjectId}/details [get]
func (h *CourseHandler) GetSubjectDetails(c *fiber.Ctx) error {
	subjectID, err := uuid.Parse(c.Params("subjectId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid subject ID",
		})
	}

	// Get current user ID
	currentUserID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "Unauthorized",
		})
	}

	// 1. Get subject info
	subject, err := h.courseService.GetSubjects(c.Context())
	if err != nil {
		return handleServiceError(c, err)
	}

	var targetSubject *courseDomain.Subject
	for _, s := range subject {
		if s.ID == subjectID {
			targetSubject = &s
			break
		}
	}

	if targetSubject == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "Subject not found",
		})
	}

	// 2. Get all courses in this subject that the user is enrolled in
	studentCourses, err := h.courseService.GetStudentCourses(c.Context(), currentUserID)
	if err != nil {
		return handleServiceError(c, err)
	}

	// Filter courses by subject
	var coursesInSubject []courseDomain.Course
	for _, course := range studentCourses {
		if course.SubjectID == subjectID {
			coursesInSubject = append(coursesInSubject, course)
		}
	}

	// 3. Build course cards with progress
	var courseCards []dto.CourseCard
	for _, course := range coursesInSubject {
		card := dto.CourseCard{
			ID:           course.ID,
			Title:        course.Title,
			Description:  course.Description,
			TeacherID:    course.TeacherID,
			DeliveryType: string(course.DeliveryType),
			LocationName: course.LocationName,
			TotalLessons: course.TotalLessons,
			Price:        course.Price,
			Currency:     course.Currency,
			IsPaid:       course.IsPaid,
			BillingType:  string(course.BillingType),
			Status:       string(course.Status),
		}

		// Get teacher info
		if h.authClient != nil {
			userInfo, err := h.authClient.GetUserInfo(c.Context(), course.TeacherID.String())
			if err == nil && userInfo != nil {
				card.TeacherName = userInfo.Name
				card.TeacherProfileImg = userInfo.ProfileImg
			}
		}

		// Get progress if available
		if h.progressService != nil {
			snapshot, err := h.progressService.GetStudentProgress(c.Context(), course.ID, currentUserID)
			if err == nil && snapshot != nil {
				totalAttended := snapshot.PresentCount + snapshot.LateCount
				attendancePercentage := 0.0
				if snapshot.TotalLessons > 0 {
					attendancePercentage = (float64(totalAttended) / float64(snapshot.TotalLessons)) * 100
				}

				status := "Good Standing"
				targetPercentage := 80.0
				if attendancePercentage < 60 {
					status = "At Risk"
				} else if attendancePercentage < targetPercentage {
					status = "Needs Improvement"
				}

				card.Progress = &dto.ProgressInfo{
					AttendancePercentage: attendancePercentage,
					ClassesAttended:      totalAttended,
					TotalClasses:         snapshot.TotalLessons,
					OverallGrade:         snapshot.OverallProgress,
					Status:               status,
					TargetPercentage:     targetPercentage,
					PresentCount:         snapshot.PresentCount,
					LateCount:            snapshot.LateCount,
					AbsentCount:          snapshot.AbsentCount,
					ExcusedCount:         snapshot.ExcusedCount,
					LastUpdated:          snapshot.CalculatedAt,
				}
			}
		}

		courseCards = append(courseCards, card)
	}

	// Build response
	response := dto.SubjectDetailsResponse{
		Subject: dto.SubjectInfo{
			ID:           targetSubject.ID,
			Name:         targetSubject.Name,
			Description:  targetSubject.Description,
			Icon:         targetSubject.Icon,
			TotalCourses: len(courseCards),
		},
		Courses: courseCards,
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    response,
	})
}


// GetCourseReviews godoc
// @Summary Get all reviews for a course
// @Tags courses
// @Produce json
// @Param id path string true "Course ID"
// @Param page query int false "Page number (default 1)"
// @Param limit query int false "Items per page (default 20, max 100)"
// @Success 200 {object} dto.CourseReviewsResponse
// @Router /api/v1/courses/{id}/reviews [get]
func (h *CourseHandler) GetCourseReviews(c *fiber.Ctx) error {
	courseID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid course ID",
		})
	}

	// Parse pagination params
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	// 1. Get course info
	course, err := h.courseService.GetCourse(c.Context(), courseID)
	if err != nil {
		return handleServiceError(c, err)
	}

	// 2. Get course average rating
	var avgRating *postgres.CourseAvgRating
	if h.courseRatingRepo != nil {
		avgRating, _ = h.courseRatingRepo.GetCourseAvgRating(c.Context(), courseID)
	}

	// 3. Get total count for pagination
	var totalCount int64
	if h.courseRatingRepo != nil {
		totalCount, _ = h.courseRatingRepo.CountCourseRatings(c.Context(), courseID)
	}

	// 4. Get paginated reviews
	var reviews []postgres.CourseRating
	if h.courseRatingRepo != nil {
		reviews, _ = h.courseRatingRepo.GetCourseRatings(c.Context(), courseID, limit, offset)
	}

	// 5. Fetch student info for each review
	var reviewResponses []dto.CourseReviewResponse
	for _, review := range reviews {
		reviewResp := dto.CourseReviewResponse{
			ID:        review.ID,
			StudentID: review.StudentID,
			Rating:    review.Rating,
			Review:    review.Review,
			CreatedAt: review.CreatedAt,
			UpdatedAt: review.UpdatedAt,
		}

		// Fetch student info from auth service
		if h.authClient != nil {
			userInfo, err := h.authClient.GetUserInfo(c.Context(), review.StudentID.String())
			if err == nil && userInfo != nil {
				reviewResp.StudentName = userInfo.Name
				reviewResp.StudentProfile = userInfo.ProfileImg
			}
		}

		reviewResponses = append(reviewResponses, reviewResp)
	}

	// Build response
	response := dto.CourseReviewsResponse{
		CourseID:    courseID,
		CourseTitle: course.Title,
		Reviews:     reviewResponses,
		Pagination: dto.PaginationInfo{
			Page:       page,
			Limit:      limit,
			TotalItems: totalCount,
			TotalPages: int((totalCount + int64(limit) - 1) / int64(limit)),
		},
	}

	// Add rating stats if available
	if avgRating != nil {
		response.AverageRating = avgRating.AvgRating
		response.TotalRatings = avgRating.TotalRatings
		response.RatingBreakdown = dto.RatingBreakdown{
			FiveStars:  avgRating.FiveStarCount,
			FourStars:  avgRating.FourStarCount,
			ThreeStars: avgRating.ThreeStarCount,
			TwoStars:   avgRating.TwoStarCount,
			OneStar:    avgRating.OneStarCount,
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    response,
	})
}

// GetCourseAssistants godoc
// @Summary Get all assistants for a course
// @Tags courses
// @Produce json
// @Param id path string true "Course ID"
// @Success 200 {array} dto.AssistantInfo
// @Router /api/v1/courses/{id}/assistants [get]
func (h *CourseHandler) GetCourseAssistants(c *fiber.Ctx) error {
	courseID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid course ID",
		})
	}

	assistants, err := h.courseService.GetCourseAssistants(c.Context(), courseID)
	if err != nil {
		return handleServiceError(c, err)
	}

	var responses []dto.AssistantInfo
	for _, asst := range assistants {
		assistantInfo := dto.AssistantInfo{
			ID:                asst.ID,
			AssistantID:       asst.AssistantID,
			CanStartLesson:    asst.CanStartLesson,
			CanEndLesson:      asst.CanEndLesson,
			CanViewAttendance: asst.CanViewAttendance,
			CanEditAttendance: asst.CanEditAttendance,
			AddedAt:           asst.CreatedAt,
		}

		// Fetch assistant user info
		if h.authClient != nil {
			userInfo, err := h.authClient.GetUserInfo(c.Context(), asst.AssistantID.String())
			if err == nil && userInfo != nil {
				assistantInfo.AssistantName = userInfo.Name
				assistantInfo.AssistantProfileImg = userInfo.ProfileImg
			}
		}

		responses = append(responses, assistantInfo)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    responses,
	})
}

// RemoveAssistant godoc
// @Summary Remove an assistant from a course
// @Tags courses
// @Produce json
// @Param id path string true "Course ID"
// @Param assistantId path string true "Assistant ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/courses/{id}/assistants/{assistantId} [delete]
func (h *CourseHandler) RemoveAssistant(c *fiber.Ctx) error {
	courseID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid course ID",
		})
	}

	assistantID, err := uuid.Parse(c.Params("assistantId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid assistant ID",
		})
	}

	teacherID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "Unauthorized",
		})
	}

	err = h.courseService.RemoveAssistant(c.Context(), courseID, teacherID, assistantID)
	if err != nil {
		return handleServiceError(c, err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Assistant removed successfully",
	})
}

// CreateCourseReview godoc
// @Summary Create a review for a course
// @Tags courses
// @Accept json
// @Produce json
// @Param id path string true "Course ID"
// @Param body body dto.CreateCourseReviewRequest true "Review data"
// @Success 201 {object} dto.CourseReviewResponse
// @Router /api/v1/courses/{id}/reviews [post]
func (h *CourseHandler) CreateCourseReview(c *fiber.Ctx) error {
	courseID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid course ID",
		})
	}

	var req dto.CreateCourseReviewRequest
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

	// Get current user ID
	studentID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "Unauthorized",
		})
	}

	// Check if course exists
	_, err = h.courseService.GetCourse(c.Context(), courseID)
	if err != nil {
		return handleServiceError(c, err)
	}

	// Check if student is enrolled and has paid
	if h.enrollmentRepo != nil {
		enrollment, err := h.enrollmentRepo.GetByCourseAndUser(c.Context(), courseID, studentID)
		if err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"error":   "Enrollment check failed: " + err.Error(),
			})
		}
		
		if enrollment == nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"error":   fmt.Sprintf("No enrollment record found for Student [%s] in Course [%s]", studentID, courseID),
			})
		}
		
		if !enrollment.IsActive {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"error":   "Your enrollment is currently inactive",
			})
		}
		
		// Ensure payment is completed for paid courses
		course, _ := h.courseService.GetCourse(c.Context(), courseID)
		if course != nil && course.IsPaid && !enrollment.IsPaid {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"error":   "PAYMENT_REQUIRED: This is a paid course and your enrollment status is still UNPAID.",
			})
		}
	}



	// Check if student already reviewed this course
	if h.courseRatingRepo != nil {
		existingReview, _ := h.courseRatingRepo.GetCourseRatingByStudent(c.Context(), courseID, studentID)
		if existingReview != nil {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"success": false,
				"error":   "You have already reviewed this course. Use PUT to update your review.",
			})
		}
	}

	// Create review
	now := time.Now()
	review := &postgres.CourseRating{
		ID:        uuid.New(),
		CourseID:  courseID,
		StudentID: studentID,
		Rating:    req.Rating,
		Review:    req.Review,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := h.courseRatingRepo.CreateCourseRating(c.Context(), review); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to create review",
		})
	}

	// Fetch student info
	response := dto.CourseReviewResponse{
		ID:        review.ID,
		StudentID: review.StudentID,
		Rating:    review.Rating,
		Review:    review.Review,
		CreatedAt: review.CreatedAt,
		UpdatedAt: review.UpdatedAt,
	}

	if h.authClient != nil {
		userInfo, _ := h.authClient.GetUserInfo(c.Context(), studentID.String())
		if userInfo != nil {
			response.StudentName = userInfo.Name
			response.StudentProfile = userInfo.ProfileImg
		}
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    response,
	})
}

// UpdateCourseReview godoc
// @Summary Update a course review
// @Tags courses
// @Accept json
// @Produce json
// @Param id path string true "Course ID"
// @Param reviewId path string true "Review ID"
// @Param body body dto.UpdateCourseReviewRequest true "Review data"
// @Success 200 {object} dto.CourseReviewResponse
// @Router /api/v1/courses/{id}/reviews/{reviewId} [put]
func (h *CourseHandler) UpdateCourseReview(c *fiber.Ctx) error {
	courseID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid course ID",
		})
	}

	reviewID, err := uuid.Parse(c.Params("reviewId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid review ID",
		})
	}

	var req dto.UpdateCourseReviewRequest
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

	// Get current user ID
	studentID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "Unauthorized",
		})
	}

	// Get existing review
	existingReview, err := h.courseRatingRepo.GetCourseRatingByStudent(c.Context(), courseID, studentID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to fetch review",
		})
	}

	if existingReview == nil || existingReview.ID != reviewID {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "Review not found or you don't have permission to edit it",
		})
	}

	// Update review
	existingReview.Rating = req.Rating
	existingReview.Review = req.Review
	existingReview.UpdatedAt = time.Now()

	if err := h.courseRatingRepo.UpdateCourseRating(c.Context(), existingReview); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to update review",
		})
	}

	// Fetch student info
	response := dto.CourseReviewResponse{
		ID:        existingReview.ID,
		StudentID: existingReview.StudentID,
		Rating:    existingReview.Rating,
		Review:    existingReview.Review,
		CreatedAt: existingReview.CreatedAt,
		UpdatedAt: existingReview.UpdatedAt,
	}

	if h.authClient != nil {
		userInfo, _ := h.authClient.GetUserInfo(c.Context(), studentID.String())
		if userInfo != nil {
			response.StudentName = userInfo.Name
			response.StudentProfile = userInfo.ProfileImg
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    response,
	})
}

// DeleteCourseReview godoc
// @Summary Delete a course review
// @Tags courses
// @Produce json
// @Param id path string true "Course ID"
// @Param reviewId path string true "Review ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/courses/{id}/reviews/{reviewId} [delete]
func (h *CourseHandler) DeleteCourseReview(c *fiber.Ctx) error {
	courseID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid course ID",
		})
	}

	reviewID, err := uuid.Parse(c.Params("reviewId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid review ID",
		})
	}

	// Get current user ID
	studentID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "Unauthorized",
		})
	}

	// Get existing review
	existingReview, err := h.courseRatingRepo.GetCourseRatingByStudent(c.Context(), courseID, studentID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to fetch review",
		})
	}

	if existingReview == nil || existingReview.ID != reviewID {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "Review not found or you don't have permission to delete it",
		})
	}

	// Delete review
	if err := h.courseRatingRepo.DeleteCourseRating(c.Context(), reviewID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to delete review",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Review deleted successfully",
	})
}
