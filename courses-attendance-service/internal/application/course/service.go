package course

import (
	"context"
	"errors"

	courseDomain "github.com/OmarrGhorab/courses-attendance-service/internal/domain/course"
	"github.com/OmarrGhorab/courses-attendance-service/internal/domain/events"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/clock"
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
)

// Service handles course-related business logic
type Service struct {
	courseRepo     *postgres.CourseRepository
	subjectRepo    *postgres.SubjectRepository
	enrollmentRepo *postgres.EnrollmentRepository
	assistantRepo  *postgres.CourseAssistantRepository
	events         *notificationevents.EventDispatcher
	clock          clock.Clock
}

func NewService(
	courseRepo *postgres.CourseRepository,
	subjectRepo *postgres.SubjectRepository,
	enrollmentRepo *postgres.EnrollmentRepository,
	assistantRepo *postgres.CourseAssistantRepository,
	events *notificationevents.EventDispatcher,
	clk clock.Clock,
) *Service {
	return &Service{
		courseRepo:     courseRepo,
		subjectRepo:    subjectRepo,
		enrollmentRepo: enrollmentRepo,
		assistantRepo:  assistantRepo,
		events:         events,
		clock:          clk,
	}
}

// CreateCourseInput represents input for creating a course
type CreateCourseInput struct {
	Title                   string
	Description             string
	SubjectID               uuid.UUID
	TeacherID               uuid.UUID
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
	AttendanceWeight        float64
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

	course := &courseDomain.Course{
		ID:                      uuid.New(),
		Title:                   input.Title,
		Description:             input.Description,
		SubjectID:               input.SubjectID,
		TeacherID:               input.TeacherID,
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
		Status:                  courseDomain.CourseStatusActive,
		AttendanceWeight:        input.AttendanceWeight,
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
	return course, nil
}

// ListCourses returns courses, optionally filtered by subject
func (s *Service) ListCourses(ctx context.Context, subjectID *uuid.UUID) ([]courseDomain.Course, error) {
	if subjectID != nil {
		return s.courseRepo.GetBySubjectID(ctx, *subjectID)
	}
	return s.courseRepo.GetAll(ctx)
}

// UpdateCourseInput represents input for updating a course
type UpdateCourseInput struct {
	Title                   *string
	Description             *string
	LocationName            *string
	LocationLat             *float64
	LocationLng             *float64
	GeofenceRadiusM         *int
	AttendanceWindowMinutes *int
	Price                   *float64
	Status                  *courseDomain.CourseStatus
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
	if input.Status != nil {
		course.Status = *input.Status
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
		return nil, ErrAlreadyEnrolled
	}

	enrollment := &courseDomain.Enrollment{
		ID:         uuid.New(),
		CourseID:   courseID,
		UserID:     studentID,
		IsActive:   true,
		IsPaid:     !course.IsPaid, // Auto-mark as paid if course is free
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

	return enrollment, nil
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
	return s.courseRepo.GetByTeacherID(ctx, teacherID)
}

// GetStudentCourses returns all courses a student is enrolled in
func (s *Service) GetStudentCourses(ctx context.Context, studentID uuid.UUID) ([]courseDomain.Course, error) {
	enrollments, err := s.enrollmentRepo.GetByUserID(ctx, studentID)
	if err != nil {
		return nil, err
	}

	var courseIDs []uuid.UUID
	for _, e := range enrollments {
		courseIDs = append(courseIDs, e.CourseID)
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

// MarkEnrollmentPaid marks an enrollment as paid
func (s *Service) MarkEnrollmentPaid(ctx context.Context, courseID, studentID uuid.UUID) error {
	enrollment, err := s.enrollmentRepo.GetByCourseAndUser(ctx, courseID, studentID)
	if err != nil {
		return err
	}
	if enrollment == nil {
		return ErrEnrollmentNotFound
	}

	now := s.clock.Now()
	enrollment.IsPaid = true
	enrollment.PaidAt = &now
	enrollment.UpdatedAt = now

	return s.enrollmentRepo.Update(ctx, enrollment)
}
