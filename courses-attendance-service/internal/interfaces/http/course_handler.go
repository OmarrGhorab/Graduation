package http

import (
	"errors"

	courseApp "github.com/OmarrGhorab/courses-attendance-service/internal/application/course"
	courseDomain "github.com/OmarrGhorab/courses-attendance-service/internal/domain/course"
	"github.com/OmarrGhorab/courses-attendance-service/internal/interfaces/http/dto"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// CourseHandler handles course-related HTTP requests
type CourseHandler struct {
	courseService *courseApp.Service
}

func NewCourseHandler(courseService *courseApp.Service) *CourseHandler {
	return &CourseHandler{courseService: courseService}
}

func (h *CourseHandler) RegisterRoutes(router fiber.Router) {
	courses := router.Group("/courses")
	courses.Post("/", h.CreateCourse)
	courses.Get("/", h.ListCourses)
	courses.Get("/:id", h.GetCourse)
	courses.Get("/my", h.GetMyCourses)
	courses.Get("/my-subjects", h.GetMySubjects)
	courses.Patch("/:id", h.UpdateCourse)
	courses.Post("/:id/enroll", h.EnrollStudent)
	courses.Post("/:id/assistants", h.AddAssistant)
	courses.Get("/:id/enrollments", h.GetCourseEnrollments)

	// Subject routes
	router.Get("/subjects", h.GetSubjects)
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
// @Success 200 {array} dto.CourseResponse
// @Router /api/v1/courses [get]
func (h *CourseHandler) ListCourses(c *fiber.Ctx) error {
	var subjectIDPtr *uuid.UUID
	subjectIDStr := c.Query("subjectId")
	if subjectIDStr != "" {
		id, err := uuid.Parse(subjectIDStr)
		if err == nil {
			subjectIDPtr = &id
		}
	}

	courses, err := h.courseService.ListCourses(c.Context(), subjectIDPtr)
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

	courses, err := h.courseService.GetStudentCourses(c.Context(), userID)
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
	subjects, err := h.courseService.GetSubjects(c.Context())
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
