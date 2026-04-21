package course

import (
	"context"
	"errors"
	"mime/multipart"

	courseDomain "github.com/OmarrGhorab/courses-attendance-service/internal/domain/course"
	"github.com/OmarrGhorab/courses-attendance-service/internal/domain/events"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/clock"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/aiclient"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/authclient"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/cloudinary"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/notificationevents"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/persistence/postgres"
	"github.com/google/uuid"
)

var (
	ErrCourseNotFound     = errors.New("course not found")
	ErrSubjectNotFound    = errors.New("subject not found")
	ErrUnauthorized       = errors.New("unauthorized to perform this action")
	ErrAlreadyEnrolled    = errors.New("student already enrolled in this course")
	ErrAssistantExists    = errors.New("assistant already added to this course")
	ErrEnrollmentNotFound = errors.New("enrollment not found")
	ErrInvalidFile        = errors.New("invalid file provided")
)

// Service handles course-related business logic
type Service struct {
	courseRepo     *postgres.CourseRepository
	subjectRepo    *postgres.SubjectRepository
	enrollmentRepo *postgres.EnrollmentRepository
	assistantRepo  *postgres.CourseAssistantRepository
	events          *notificationevents.EventDispatcher
	clock           clock.Clock
	teacherRatingRepo *postgres.TeacherRatingRepository
	progressRepo      *postgres.ProgressSnapshotRepository
	authClient        *authclient.Client
	aiClient          *aiclient.Client
	cloudinaryClient  *cloudinary.Client
}

// ChildAnalytics represents a summary of a child's progress for a parent
type ChildAnalytics struct {
	ChildID         uuid.UUID        `json:"childId"`
	Name            string           `json:"name"`
	ProfileImg      string           `json:"profileImg"`
	EnrolledCourses int              `json:"enrolledCourses"`
	Courses         []CourseProgress `json:"courses"`
}

// CourseProgress represents a child's details in a specific course
type CourseProgress struct {
	CourseID        uuid.UUID `json:"courseId"`
	Title           string    `json:"title"`
	TeacherName     string    `json:"teacherName"`
	TeacherRating   float64   `json:"teacherRating"`
	Progress        float64   `json:"progress"`         // Overall progress %
	Attendance      float64   `json:"attendance"`       // Attendance %
	LessonsCompleted int       `json:"lessonsCompleted"`
	TotalLessons    int       `json:"totalLessons"`
	Status          string    `json:"status"`           // Course status
}

// TeacherAnalytics represents the top-level stats for a teacher
type TeacherAnalytics struct {
	TotalCourses      int               `json:"totalCourses"`
	TotalStudents     int               `json:"totalStudents"`
	TotalActiveShared int               `json:"totalActiveShared"` // Students across all courses
	TotalRevenue      float64           `json:"totalRevenue"`
	TotalAssistants   int               `json:"totalAssistants"`
	AverageRating     float64           `json:"averageRating"`
	CourseBreakdown   []CourseBreakdown `json:"courseBreakdown"`
}

// CourseBreakdown represents stats for a single course in the teacher's dashboard
type CourseBreakdown struct {
	CourseID       uuid.UUID `json:"courseId"`
	Title          string    `json:"title"`
	StudentCount   int       `json:"studentCount"`
	AssistantCount int       `json:"assistantCount"`
	Revenue        float64   `json:"revenue"`
	DeliveryType   string    `json:"deliveryType"`
	Status         string    `json:"status"`
}

// ChatContexts represents all relevant relationships for chat discovery
type ChatContexts struct {
	Teachers   []uuid.UUID `json:"teachers"`
	Students   []uuid.UUID `json:"students"`
	Assistants []uuid.UUID `json:"assistants"`
	Groups     []GroupInfo `json:"groups"`
}

// GroupInfo represents a course-based group
type GroupInfo struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Image string    `json:"image"`
}

func NewService(
	courseRepo *postgres.CourseRepository,
	subjectRepo *postgres.SubjectRepository,
	enrollmentRepo *postgres.EnrollmentRepository,
	assistantRepo *postgres.CourseAssistantRepository,
	events *notificationevents.EventDispatcher,
	teacherRatingRepo *postgres.TeacherRatingRepository,
	progressRepo *postgres.ProgressSnapshotRepository,
	authClient *authclient.Client,
	aiClient *aiclient.Client,
	cloudinaryClient *cloudinary.Client,
	clk clock.Clock,
) *Service {
	return &Service{
		courseRepo:        courseRepo,
		subjectRepo:       subjectRepo,
		enrollmentRepo:    enrollmentRepo,
		assistantRepo:     assistantRepo,
		events:            events,
		teacherRatingRepo: teacherRatingRepo,
		progressRepo:      progressRepo,
		authClient:        authClient,
		aiClient:          aiClient,
		cloudinaryClient:  cloudinaryClient,
		clock:             clk,
	}
}

// CreateCourseInput represents input for creating a course
type CreateCourseInput struct {
	Title                   string
	Description             string
	SubjectID               uuid.UUID
	TeacherID               uuid.UUID
	CourseImage             string
	GroupImage              string
	DeliveryType            courseDomain.DeliveryType
	LocationName            string
	LocationLat             *float64
	LocationLng             *float64
	GeofenceRadiusM         int
	TotalLessons            int
	AttendanceWindowMinutes int
	Price                   float64
	Currency                string
	IsPaid                  bool
	BillingType             courseDomain.BillingType
	FreeTrialLessons        int
	AttendanceWeight        float64
	PreviewVideoURL         string
	PreviewVideoPublicID    string
	ReminderIntervals       string
}

// CreateCourse creates a new course
func (s *Service) CreateCourse(ctx context.Context, input CreateCourseInput) (*courseDomain.Course, error) {
	// Verify subject exists
	subject, err := s.subjectRepo.GetByID(ctx, input.SubjectID)
	if err != nil {
		return nil, err
	}
	if subject == nil {
		return nil, ErrSubjectNotFound
	}

	// Set defaults
	if input.AttendanceWindowMinutes == 0 {
		input.AttendanceWindowMinutes = 15
	}
	if input.AttendanceWeight == 0 {
		input.AttendanceWeight = 0.30
	}
	if input.GeofenceRadiusM == 0 {
		input.GeofenceRadiusM = 100
	}
	if input.Currency == "" {
		input.Currency = "EGP"
	}
	if input.BillingType == "" {
		input.BillingType = courseDomain.BillingTypeOneTime
	}
	if input.ReminderIntervals == "" {
		input.ReminderIntervals = "15,10,5"
	}

	course := &courseDomain.Course{
		ID:                      uuid.New(),
		Title:                   input.Title,
		Description:             input.Description,
		SubjectID:               input.SubjectID,
		TeacherID:               input.TeacherID,
		CourseImage:             input.CourseImage,
		GroupImage:              input.GroupImage,
		DeliveryType:            input.DeliveryType,
		LocationName:            input.LocationName,
		LocationLat:             input.LocationLat,
		LocationLng:             input.LocationLng,
		GeofenceRadiusM:         input.GeofenceRadiusM,
		TotalLessons:            input.TotalLessons,
		AttendanceWindowMinutes: input.AttendanceWindowMinutes,
		Price:                   input.Price,
		Currency:                input.Currency,
		IsPaid:                  input.IsPaid,
		BillingType:             input.BillingType,
		FreeTrialLessons:        input.FreeTrialLessons,
		Status:                  courseDomain.CourseStatusActive,
		AttendanceWeight:        input.AttendanceWeight,
		PreviewVideoURL:         input.PreviewVideoURL,
		PreviewVideoPublicID:    input.PreviewVideoPublicID,
		ReminderIntervals:       input.ReminderIntervals,
		CreatedAt:               s.clock.Now(),
		UpdatedAt:               s.clock.Now(),
	}

	if err := s.courseRepo.Create(ctx, course); err != nil {
		return nil, err
	}

	return course, nil
}

// GetCourse retrieves a course by ID
func (s *Service) GetCourse(ctx context.Context, id uuid.UUID) (*courseDomain.Course, error) {
	course, err := s.courseRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if course == nil {
		return nil, ErrCourseNotFound
	}
	
	count, _ := s.enrollmentRepo.CountByCourseID(ctx, id)
	course.EnrollmentCount = int(count)
	
	return course, nil
}

// ListCourses returns courses with filtering and pagination
func (s *Service) ListCourses(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]courseDomain.Course, int64, error) {
	courses, total, err := s.courseRepo.ListCoursesWithFilters(ctx, filters, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	
	// Populate counts for all courses
	for i := range courses {
		count, _ := s.enrollmentRepo.CountByCourseID(ctx, courses[i].ID)
		courses[i].EnrollmentCount = int(count)
	}
	
	return courses, total, nil
}

// UpdateCourseInput represents input for updating a course
type UpdateCourseInput struct {
	Title                   *string
	Description             *string
	CourseImage             *string
	GroupImage              *string
	LocationName            *string
	LocationLat             *float64
	LocationLng             *float64
	GeofenceRadiusM         *int
	AttendanceWindowMinutes *int
	Price                   *float64
	FreeTrialLessons        *int
	BillingType             *courseDomain.BillingType
	Status                  *courseDomain.CourseStatus
	PreviewVideoURL         *string
	PreviewVideoPublicID    *string
	ReminderIntervals       *string
}

// UpdateCourse updates a course (teacher only)
func (s *Service) UpdateCourse(ctx context.Context, courseID uuid.UUID, teacherID uuid.UUID, input UpdateCourseInput) (*courseDomain.Course, error) {
	course, err := s.courseRepo.GetByID(ctx, courseID)
	if err != nil {
		return nil, err
	}
	if course == nil {
		return nil, ErrCourseNotFound
	}
	if course.TeacherID != teacherID {
		return nil, ErrUnauthorized
	}

	// Apply updates
	if input.Title != nil {
		course.Title = *input.Title
	}
	if input.Description != nil {
		course.Description = *input.Description
	}
	if input.CourseImage != nil {
		course.CourseImage = *input.CourseImage
	}
	if input.GroupImage != nil {
		course.GroupImage = *input.GroupImage
	}
	if input.LocationName != nil {
		course.LocationName = *input.LocationName
	}
	if input.LocationLat != nil {
		course.LocationLat = input.LocationLat
	}
	if input.LocationLng != nil {
		course.LocationLng = input.LocationLng
	}
	if input.GeofenceRadiusM != nil {
		course.GeofenceRadiusM = *input.GeofenceRadiusM
	}
	if input.AttendanceWindowMinutes != nil {
		course.AttendanceWindowMinutes = *input.AttendanceWindowMinutes
	}
	if input.Price != nil {
		course.Price = *input.Price
	}
	if input.FreeTrialLessons != nil {
		course.FreeTrialLessons = *input.FreeTrialLessons
	}
	if input.BillingType != nil {
		course.BillingType = *input.BillingType
	}
	if input.Status != nil {
		course.Status = *input.Status
	}
	if input.PreviewVideoURL != nil {
		course.PreviewVideoURL = *input.PreviewVideoURL
	}
	if input.PreviewVideoPublicID != nil {
		course.PreviewVideoPublicID = *input.PreviewVideoPublicID
	}
	if input.ReminderIntervals != nil {
		course.ReminderIntervals = *input.ReminderIntervals
	}
	course.UpdatedAt = s.clock.Now()

	if err := s.courseRepo.Update(ctx, course); err != nil {
		return nil, err
	}

	return course, nil
}

// EnrollStudent enrolls a student in a course
func (s *Service) EnrollStudent(ctx context.Context, courseID, studentID uuid.UUID) (*courseDomain.Enrollment, error) {
	// Verify course exists
	course, err := s.courseRepo.GetByID(ctx, courseID)
	if err != nil {
		return nil, err
	}
	if course == nil {
		return nil, ErrCourseNotFound
	}

	// Check if already enrolled
	existing, err := s.enrollmentRepo.GetByCourseAndUser(ctx, courseID, studentID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		// If already paid, it's a real error. If not paid, just return the existing one for idempotency.
		if existing.IsPaid {
			return nil, ErrAlreadyEnrolled
		}
		return existing, nil
	}

	enrollment := &courseDomain.Enrollment{
		ID:         uuid.New(),
		CourseID:   courseID,
		UserID:     studentID,
		IsActive:   !course.IsPaid, // Only active by default if the course is free
		IsPaid:     !course.IsPaid,
		EnrolledAt: s.clock.Now(),
		UpdatedAt:  s.clock.Now(),
	}

	if err := s.enrollmentRepo.Create(ctx, enrollment); err != nil {
		return nil, err
	}

	// Emit notification event for teacher
	s.events.EmitNotificationRequested(ctx, studentID, events.NotificationRequestedPayload{
		RecipientID: course.TeacherID,
		Type:        "COURSE_ENROLLMENT",
		Data: map[string]interface{}{
			"course_name": course.Title,
			"student_id":  studentID.String(),
		},
	})

	// Pre-trigger recommendation refresh in background
	if s.aiClient != nil {
		go s.aiClient.InvalidateRecommendationCache(context.Background(), studentID.String())
	}

	return enrollment, nil
}


func (s *Service) GetEnrollment(ctx context.Context, courseID, studentID uuid.UUID) (*courseDomain.Enrollment, error) {
	return s.enrollmentRepo.GetByCourseAndUser(ctx, courseID, studentID)
}


// AddAssistant adds an assistant to a course
func (s *Service) AddAssistant(ctx context.Context, courseID, teacherID, assistantID uuid.UUID) (*courseDomain.CourseAssistant, error) {
	// Verify course and ownership
	course, err := s.courseRepo.GetByID(ctx, courseID)
	if err != nil {
		return nil, err
	}
	if course == nil {
		return nil, ErrCourseNotFound
	}
	if course.TeacherID != teacherID {
		return nil, ErrUnauthorized
	}

	// Check if assistant already exists
	existing, err := s.assistantRepo.GetByCourseAndAssistant(ctx, courseID, assistantID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrAssistantExists
	}

	assistant := &courseDomain.CourseAssistant{
		ID:                uuid.New(),
		CourseID:          courseID,
		AssistantID:       assistantID,
		CanStartLesson:    true,
		CanEndLesson:      true,
		CanViewAttendance: true,
		CanEditAttendance: false,
		CreatedAt:         s.clock.Now(),
	}

	if err := s.assistantRepo.Create(ctx, assistant); err != nil {
		return nil, err
	}

	return assistant, nil
}

// GetSubjects returns all available subjects
func (s *Service) GetSubjects(ctx context.Context) ([]courseDomain.Subject, error) {
	return s.subjectRepo.GetAll(ctx)
}

// GetTeacherCourses returns all courses for a teacher
func (s *Service) GetTeacherCourses(ctx context.Context, teacherID uuid.UUID) ([]courseDomain.Course, error) {
	courses, err := s.courseRepo.GetByTeacherID(ctx, teacherID)
	if err != nil {
		return nil, err
	}

	for i := range courses {
		count, _ := s.enrollmentRepo.CountByCourseID(ctx, courses[i].ID)
		courses[i].EnrollmentCount = int(count)
	}

	return courses, nil
}

// GetStudentCourses returns all courses a student is enrolled in
func (s *Service) GetStudentCourses(ctx context.Context, studentID uuid.UUID) ([]courseDomain.Course, error) {
	enrollments, err := s.enrollmentRepo.GetByUserID(ctx, studentID)
	if err != nil {
		return nil, err
	}

	var courseIDs []uuid.UUID
	for _, e := range enrollments {
		if e.IsActive {
			courseIDs = append(courseIDs, e.CourseID)
		}
	}

	if len(courseIDs) == 0 {
		return []courseDomain.Course{}, nil
	}

	return s.courseRepo.GetByIDs(ctx, courseIDs)
}

// GetStudentSubjects returns all subjects a student is enrolled in through courses
func (s *Service) GetStudentSubjects(ctx context.Context, studentID uuid.UUID) ([]courseDomain.Subject, error) {
	courses, err := s.GetStudentCourses(ctx, studentID)
	if err != nil {
		return nil, err
	}

	subjectMap := make(map[uuid.UUID]courseDomain.Subject)
	for _, c := range courses {
		if c.Subject != nil {
			subjectMap[c.SubjectID] = *c.Subject
		}
	}

	var subjects []courseDomain.Subject
	for _, sub := range subjectMap {
		subjects = append(subjects, sub)
	}

	return subjects, nil
}

// GetCourseEnrollments returns all enrollments for a course
func (s *Service) GetCourseEnrollments(ctx context.Context, courseID uuid.UUID) ([]courseDomain.Enrollment, error) {
	return s.enrollmentRepo.GetByCourseID(ctx, courseID)
}

// IsEnrolled checks if a student is enrolled in a course
func (s *Service) IsEnrolled(ctx context.Context, courseID, studentID uuid.UUID) (bool, error) {
	enrollment, err := s.enrollmentRepo.GetByCourseAndUser(ctx, courseID, studentID)
	if err != nil {
		return false, err
	}
	return enrollment != nil && enrollment.IsActive, nil
}

// IsTeacherOrAssistant checks if a user is the teacher or an assistant for a course
func (s *Service) IsTeacherOrAssistant(ctx context.Context, courseID, userID uuid.UUID) (bool, bool, error) {
	course, err := s.courseRepo.GetByID(ctx, courseID)
	if err != nil {
		return false, false, err
	}
	if course == nil {
		return false, false, ErrCourseNotFound
	}

	if course.TeacherID == userID {
		return true, false, nil
	}

	assistant, err := s.assistantRepo.GetByCourseAndAssistant(ctx, courseID, userID)
	if err != nil {
		return false, false, err
	}
	if assistant != nil {
		return false, true, nil
	}

	return false, false, nil
}

// MarkEnrollmentPaid marks an enrollment as paid, optionally for a specific period (YYYY-MM)
func (s *Service) MarkEnrollmentPaid(ctx context.Context, courseID, studentID uuid.UUID, periodKey string) error {
	enrollment, err := s.enrollmentRepo.GetByCourseAndUser(ctx, courseID, studentID)
	if err != nil {
		return err
	}

	now := s.clock.Now()

	if enrollment == nil {
		// Auto-enroll if missing
		course, err := s.courseRepo.GetByID(ctx, courseID)
		if err != nil || course == nil {
			return errors.New("course not found")
		}

		enrollment = &courseDomain.Enrollment{
			ID:         uuid.New(),
			CourseID:   courseID,
			UserID:     studentID,
			IsActive:   true,
			IsPaid:     true,
			PaidAt:     &now,
			EnrolledAt: now,
			UpdatedAt:  now,
		}
		if err := s.enrollmentRepo.Create(ctx, enrollment); err != nil {
			return err
		}
	} else {
		enrollment.IsPaid = true
		enrollment.IsActive = true // Explicitly activate upon payment
		enrollment.PaidAt = &now
		enrollment.UpdatedAt = now
		if err := s.enrollmentRepo.Update(ctx, enrollment); err != nil {
			return err
		}
	}

	// If a specific period is provided (for monthly billing), mark it as paid
	if periodKey != "" {
		existingPeriod, err := s.enrollmentRepo.GetPeriod(ctx, enrollment.ID, periodKey)
		if err != nil {
			return err
		}

		if existingPeriod == nil {
			period := &courseDomain.EnrollmentPeriod{
				ID:           uuid.New(),
				EnrollmentID: enrollment.ID,
				PeriodKey:    periodKey,
				IsPaid:       true,
				PaidAt:       &now,
				CreatedAt:    now,
				UpdatedAt:    now,
			}
			return s.enrollmentRepo.CreatePeriod(ctx, period)
		} else {
			existingPeriod.IsPaid = true
			existingPeriod.PaidAt = &now
			existingPeriod.UpdatedAt = now
			return s.enrollmentRepo.UpdatePeriod(ctx, existingPeriod)
		}
	}

	return nil
}



// GetCourseAssistants returns all assistants for a course
func (s *Service) GetCourseAssistants(ctx context.Context, courseID uuid.UUID) ([]courseDomain.CourseAssistant, error) {
	return s.assistantRepo.GetByCourseID(ctx, courseID)
}

// RemoveAssistant removes an assistant from a course
func (s *Service) RemoveAssistant(ctx context.Context, courseID, teacherID, assistantID uuid.UUID) error {
	// Verify course and ownership
	course, err := s.courseRepo.GetByID(ctx, courseID)
	if err != nil {
		return err
	}
	if course == nil {
		return ErrCourseNotFound
	}
	if course.TeacherID != teacherID {
		return ErrUnauthorized
	}

	// Get assistant to verify it exists
	assistant, err := s.assistantRepo.GetByCourseAndAssistant(ctx, courseID, assistantID)
	if err != nil {
		return err
	}
	if assistant == nil {
		return errors.New("assistant not found")
	}

	return s.assistantRepo.Delete(ctx, courseID, assistantID)
}


func (s *Service) ListSubjects(ctx context.Context) ([]courseDomain.Subject, error) {
	return s.subjectRepo.GetAll(ctx)
}

func (s *Service) CreateSubject(ctx context.Context, name, description, icon string) (*courseDomain.Subject, error) {
	subject := &courseDomain.Subject{
		ID:          uuid.New(),
		Name:        name,
		Description: description,
		Icon:        icon,
		CreatedAt:   s.clock.Now(),
		UpdatedAt:   s.clock.Now(),
	}

	if err := s.subjectRepo.Create(ctx, subject); err != nil {
		return nil, err
	}

	return subject, nil
}


// GetTeacherAnalytics aggregates all analytics for a teacher
func (s *Service) GetTeacherAnalytics(ctx context.Context, teacherID uuid.UUID) (*TeacherAnalytics, error) {
	// 1. Get all courses for this teacher
	courses, err := s.courseRepo.GetByTeacherID(ctx, teacherID)
	if err != nil {
		return nil, err
	}

	analytics := &TeacherAnalytics{
		TotalCourses:    len(courses),
		CourseBreakdown: []CourseBreakdown{},
	}

	totalStudentSet := make(map[uuid.UUID]bool)
	assistantSet := make(map[uuid.UUID]bool)

	for _, c := range courses {
		// Get enrollments for this course
		enrollments, err := s.enrollmentRepo.GetByCourseID(ctx, c.ID)
		if err != nil {
			continue
		}

		studentCount := len(enrollments)
		analytics.TotalStudents += studentCount

		// Unique students across all courses
		for _, e := range enrollments {
			totalStudentSet[e.UserID] = true
		}

		// Calculate revenue for this course
		courseRevenue := 0.0
		for _, e := range enrollments {
			if e.IsPaid {
				switch c.BillingType {
				case courseDomain.BillingTypeOneTime:
					courseRevenue += c.Price
				case courseDomain.BillingTypeMonthly:
					// For monthly, we need to count individual paid periods
					periods, _ := s.enrollmentRepo.GetPeriods(ctx, e.ID)
					courseRevenue += c.Price * float64(len(periods))
				}
			}
		}
		analytics.TotalRevenue += courseRevenue

		// Get assistants for this course
		assistants, err := s.assistantRepo.GetByCourseID(ctx, c.ID)
		if err == nil {
			for _, a := range assistants {
				assistantSet[a.AssistantID] = true
			}
		}

		analytics.CourseBreakdown = append(analytics.CourseBreakdown, CourseBreakdown{
			CourseID:       c.ID,
			Title:          c.Title,
			StudentCount:   studentCount,
			AssistantCount: len(assistants),
			Revenue:        courseRevenue,
			DeliveryType:   string(c.DeliveryType),
			Status:         string(c.Status),
		})
	}

	analytics.TotalActiveShared = len(totalStudentSet)
	analytics.TotalAssistants = len(assistantSet)

	// Get teacher's average rating
	rating, _ := s.teacherRatingRepo.GetTeacherAvgRating(ctx, teacherID)
	if rating != nil {
		analytics.AverageRating = rating.AvgRating
	}

	return analytics, nil
}

// GetParentAnalytics gets analytics for all children of a parent
func (s *Service) GetParentAnalytics(ctx context.Context, parentID uuid.UUID) ([]ChildAnalytics, error) {
	// 1. Get all children linked to this parent from auth service
	children, err := s.authClient.GetChildren(ctx, parentID.String())
	if err != nil {
		return nil, err
	}

	result := []ChildAnalytics{}

	for _, childInfo := range children {
		childID, err := uuid.Parse(childInfo.ID)
		if err != nil {
			continue
		}

		// 2. Get enrollments for this child
		enrollments, err := s.enrollmentRepo.GetByUserID(ctx, childID)
		if err != nil {
			continue
		}

		childAnalytic := ChildAnalytics{
			ChildID:         childID,
			Name:            childInfo.Name,
			ProfileImg:      childInfo.ProfileImg,
			EnrolledCourses: len(enrollments),
			Courses:         []CourseProgress{},
		}

		for _, e := range enrollments {
			// Get course info
			course, err := s.courseRepo.GetByID(ctx, e.CourseID)
			if err != nil || course == nil {
				continue
			}

			// Get progress snapshot
			progress, err := s.progressRepo.GetByCourseAndStudent(ctx, e.CourseID, childID)
			
			// Get teacher info
			var teacherName string
			var teacherRating float64
			teacherInfo, err := s.authClient.GetUserInfo(ctx, course.TeacherID.String())
			if err == nil && teacherInfo != nil {
				teacherName = teacherInfo.Name
			}
			
			rating, _ := s.teacherRatingRepo.GetTeacherAvgRating(ctx, course.TeacherID)
			if rating != nil {
				teacherRating = rating.AvgRating
			}

			cp := CourseProgress{
				CourseID:      course.ID,
				Title:         course.Title,
				TeacherName:   teacherName,
				TeacherRating: teacherRating,
				Status:        string(course.Status),
			}

			if progress != nil {
				cp.Progress = progress.OverallProgress
				cp.Attendance = progress.AttendanceRatio * 100
				cp.LessonsCompleted = progress.CompletedLessons
				cp.TotalLessons = progress.TotalLessons
			}

			childAnalytic.Courses = append(childAnalytic.Courses, cp)
		}

		result = append(result, childAnalytic)
	}

	return result, nil
}

// GetChatContexts returns all academic relationships for a user
func (s *Service) GetChatContexts(ctx context.Context, userID uuid.UUID, role string) (*ChatContexts, error) {
	contexts := &ChatContexts{
		Teachers:   []uuid.UUID{},
		Students:   []uuid.UUID{},
		Assistants: []uuid.UUID{},
		Groups:     []GroupInfo{},
	}

	switch role {
	case "STUDENT":
		// Get enrolled courses
		courses, err := s.GetStudentCourses(ctx, userID)
		if err != nil {
			return nil, err
		}

		teacherSet := make(map[uuid.UUID]bool)
		assistantSet := make(map[uuid.UUID]bool)

		for _, c := range courses {
			teacherSet[c.TeacherID] = true
			
			// Priority: GroupImage -> CourseImage
			img := c.GroupImage
			if img == "" {
				img = c.CourseImage
			}
			contexts.Groups = append(contexts.Groups, GroupInfo{
				ID:    c.ID,
				Name:  c.Title,
				Image: img,
			})

			// Get assistants for this course
			assistants, _ := s.GetCourseAssistants(ctx, c.ID)
			for _, a := range assistants {
				assistantSet[a.AssistantID] = true
			}
		}

		for tID := range teacherSet {
			contexts.Teachers = append(contexts.Teachers, tID)
		}
		for aID := range assistantSet {
			contexts.Assistants = append(contexts.Assistants, aID)
		}

	case "TEACHER", "INSTRUCTOR":
		// Get taught courses
		courses, err := s.GetTeacherCourses(ctx, userID)
		if err != nil {
			return nil, err
		}

		studentSet := make(map[uuid.UUID]bool)
		assistantSet := make(map[uuid.UUID]bool)

		for _, c := range courses {
			// Priority: GroupImage -> CourseImage
			img := c.GroupImage
			if img == "" {
				img = c.CourseImage
			}
			contexts.Groups = append(contexts.Groups, GroupInfo{
				ID:    c.ID, 
				Name:  c.Title,
				Image: img,
			})

			// Get students
			enrollments, _ := s.GetCourseEnrollments(ctx, c.ID)
			for _, e := range enrollments {
				studentSet[e.UserID] = true
			}

			// Get assistants
			assistants, _ := s.GetCourseAssistants(ctx, c.ID)
			for _, a := range assistants {
				assistantSet[a.AssistantID] = true
			}
		}

		for sID := range studentSet {
			contexts.Students = append(contexts.Students, sID)
		}
		for aID := range assistantSet {
			contexts.Assistants = append(contexts.Assistants, aID)
		}

	case "ASSISTANT":
		// Get all courses where this user is an assistant
		assistedCourseIDs, err := s.assistantRepo.GetCoursesByAssistantID(ctx, userID)
		if err != nil {
			return nil, err
		}

		if len(assistedCourseIDs) > 0 {
			courses, _ := s.courseRepo.GetByIDs(ctx, assistedCourseIDs)
			studentSet := make(map[uuid.UUID]bool)
			teacherSet := make(map[uuid.UUID]bool)

			for _, c := range courses {
				// Priority: GroupImage -> CourseImage
				img := c.GroupImage
				if img == "" {
					img = c.CourseImage
				}
				contexts.Groups = append(contexts.Groups, GroupInfo{
					ID:    c.ID, 
					Name:  c.Title,
					Image: img,
				})
				teacherSet[c.TeacherID] = true

				// Get students
				enrollments, _ := s.GetCourseEnrollments(ctx, c.ID)
				for _, e := range enrollments {
					studentSet[e.UserID] = true
				}
			}

			for sID := range studentSet {
				contexts.Students = append(contexts.Students, sID)
			}
			for tID := range teacherSet {
				contexts.Teachers = append(contexts.Teachers, tID)
			}
		}
	}

	return contexts, nil
}

func (s *Service) GetTeacherAuthority(ctx context.Context, teacherID uuid.UUID) (int, error) {
	count, err := s.enrollmentRepo.CountByTeacherID(ctx, teacherID)
	return int(count), err
}

// UploadCourseImage uploads a course image to Cloudinary and returns the URL
func (s *Service) UploadCourseImage(ctx context.Context, teacherID uuid.UUID, file multipart.File, filename string) (string, error) {
	result, err := s.cloudinaryClient.UploadImage(ctx, file, filename)
	if err != nil {
		return "", err
	}

	return result.URL, nil
}

// UploadCourseVideo uploads a course preview video to Cloudinary and returns the URL and PublicID
func (s *Service) UploadCourseVideo(ctx context.Context, teacherID uuid.UUID, file multipart.File, filename string) (string, string, error) {
	result, err := s.cloudinaryClient.UploadVideo(ctx, file, filename)
	if err != nil {
		return "", "", err
	}

	url := result.StreamingURL
	if url == "" {
		url = result.URL
	}

	return url, result.PublicID, nil
}
