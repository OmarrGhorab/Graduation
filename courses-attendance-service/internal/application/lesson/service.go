package lesson

import (
	"context"
	"errors"
	"time"

	progressApp "github.com/OmarrGhorab/courses-attendance-service/internal/application/progress"
	"github.com/OmarrGhorab/courses-attendance-service/internal/domain/events"
	lessonDomain "github.com/OmarrGhorab/courses-attendance-service/internal/domain/lesson"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/clock"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/notificationevents"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/persistence/postgres"
	"github.com/google/uuid"
)

var (
	ErrLessonNotFound   = errors.New("lesson not found")
	ErrCourseNotFound   = errors.New("course not found")
	ErrUnauthorized     = errors.New("unauthorized to perform this action")
	ErrInvalidStatus    = errors.New("invalid lesson status for this operation")
	ErrLessonInProgress = errors.New("lesson is currently in progress")
	ErrLessonNotLive    = errors.New("lesson is not currently live")
)

// Service handles lesson-related business logic
type Service struct {
	lessonRepo      *postgres.LessonRepository
	courseRepo      *postgres.CourseRepository
	enrollmentRepo  *postgres.EnrollmentRepository
	progressService *progressApp.Service
	events          *notificationevents.EventDispatcher
	clock           clock.Clock
}


func NewService(
	lessonRepo *postgres.LessonRepository,
	courseRepo *postgres.CourseRepository,
	enrollmentRepo *postgres.EnrollmentRepository,
	progressService *progressApp.Service,
	events *notificationevents.EventDispatcher,
	clk clock.Clock,
) *Service {
	return &Service{
		lessonRepo:      lessonRepo,
		courseRepo:      courseRepo,
		enrollmentRepo:  enrollmentRepo,
		progressService: progressService,
		events:          events,
		clock:           clk,
	}
}


// CreateLessonInput represents input for creating a lesson
type CreateLessonInput struct {
	CourseID        uuid.UUID
	Title           string
	Description     string
	ScheduledAt     time.Time
	DurationMinutes int
	DeliveryType    lessonDomain.DeliveryType
	IsFree          bool
	
	// Online lesson materials
	VideoURL      string
	VideoPublicID string
	MaterialsURL  string
	Duration      *int
	
	// Location (for OFFLINE)
	LocationName    string
	LocationLat     *float64
	LocationLng     *float64
	GeofenceRadiusM *int
}

// CreateLesson creates a new lesson
func (s *Service) CreateLesson(ctx context.Context, teacherID uuid.UUID, input CreateLessonInput) (*lessonDomain.Lesson, error) {
	// Verify course and ownership
	course, err := s.courseRepo.GetByID(ctx, input.CourseID)
	if err != nil {
		return nil, err
	}
	if course == nil {
		return nil, ErrCourseNotFound
	}
	if course.TeacherID != teacherID {
		return nil, ErrUnauthorized
	}

	// Get next lesson number
	count, err := s.lessonRepo.CountByCourse(ctx, input.CourseID)
	if err != nil {
		return nil, err
	}
	lessonNumber := int(count) + 1

	// Set defaults
	if input.DurationMinutes == 0 {
		input.DurationMinutes = 60
	}

	lesson := &lessonDomain.Lesson{
		ID:              uuid.New(),
		CourseID:        input.CourseID,
		Title:           input.Title,
		Description:     input.Description,
		LessonNumber:    lessonNumber,
		ScheduledAt:     input.ScheduledAt.UTC(),
		DurationMinutes: input.DurationMinutes,
		Status:          lessonDomain.LessonStatusScheduled,
		DeliveryType:    input.DeliveryType,
		IsFree:          input.IsFree,
		VideoURL:        input.VideoURL,
		VideoPublicID:   input.VideoPublicID,
		MaterialsURL:    input.MaterialsURL,
		Duration:        input.Duration,
		LocationName:    input.LocationName,
		LocationLat:     input.LocationLat,
		LocationLng:     input.LocationLng,
		GeofenceRadiusM: input.GeofenceRadiusM,
		CreatedAt:       s.clock.Now(),
		UpdatedAt:       s.clock.Now(),
	}

	if err := s.lessonRepo.Create(ctx, lesson); err != nil {
		return nil, err
	}

	// Update course total lessons count
	course.TotalLessons = lessonNumber
	course.UpdatedAt = s.clock.Now()
	if err := s.courseRepo.Update(ctx, course); err != nil {
		return nil, err
	}

	return lesson, nil
}

// GetLesson retrieves a lesson by ID
func (s *Service) GetLesson(ctx context.Context, id uuid.UUID) (*lessonDomain.Lesson, error) {
	lesson, err := s.lessonRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if lesson == nil {
		return nil, ErrLessonNotFound
	}
	return lesson, nil
}

// GetCourseLessons retrieves all lessons for a course, filtered by payment periods for students
func (s *Service) GetCourseLessons(ctx context.Context, courseID uuid.UUID, userID uuid.UUID, userRole string) ([]lessonDomain.Lesson, error) {
	lessons, err := s.lessonRepo.GetByCourseID(ctx, courseID)
	if err != nil {
		return nil, err
	}

	// Teachers and Instructors see everything
	if userRole == "TEACHER" || userRole == "INSTRUCTOR" || userRole == "ADMIN" {
		return lessons, nil
	}

	// Check if this course has monthly billing
	course, err := s.courseRepo.GetByID(ctx, courseID)
	if err != nil || course == nil {
		return lessons, nil // Fallback to all if course info missing
	}

	if course.BillingType != "MONTHLY" {
		return lessons, nil // One-time courses allow access to all content
	}

	// For students in monthly courses, filter content by paid periods
	enrollment, err := s.enrollmentRepo.GetByCourseAndUser(ctx, courseID, userID)
	if err != nil || enrollment == nil {
		return []lessonDomain.Lesson{}, nil
	}

	paidPeriods, err := s.enrollmentRepo.GetPeriods(ctx, enrollment.ID)
	if err != nil {
		return nil, err
	}

	paidMap := make(map[string]bool)
	for _, p := range paidPeriods {
		paidMap[p.PeriodKey] = true
	}

	var filtered []lessonDomain.Lesson
	for _, l := range lessons {
		// Free lessons are always visible
		if l.IsFree {
			filtered = append(filtered, l)
			continue
		}

		// Check if the lesson's month is paid
		periodKey := l.ScheduledAt.Format("2006-01")
		if paidMap[periodKey] {
			filtered = append(filtered, l)
		}
	}

	return filtered, nil
}

// StartLesson starts a lesson (sets status to LIVE)
func (s *Service) StartLesson(ctx context.Context, lessonID uuid.UUID) (*lessonDomain.Lesson, error) {
	lesson, err := s.lessonRepo.GetByID(ctx, lessonID)
	if err != nil {
		return nil, err
	}
	if lesson == nil {
		return nil, ErrLessonNotFound
	}

	if lesson.Status != lessonDomain.LessonStatusScheduled {
		return nil, ErrInvalidStatus
	}

	now := s.clock.Now()
	lesson.Status = lessonDomain.LessonStatusLive
	lesson.StartsAt = &now
	lesson.UpdatedAt = now

	if err := s.lessonRepo.Update(ctx, lesson); err != nil {
		return nil, err
	}

	// Emit event
	s.events.EmitLessonStarted(ctx, events.LessonStartedPayload{
		LessonID: lesson.ID,
		CourseID: lesson.CourseID,
		StartsAt: *lesson.StartsAt,
	})

	return lesson, nil
}

// EndLesson ends a lesson (sets status to COMPLETED)
func (s *Service) EndLesson(ctx context.Context, lessonID uuid.UUID) (*lessonDomain.Lesson, error) {
	lesson, err := s.lessonRepo.GetByID(ctx, lessonID)
	if err != nil {
		return nil, err
	}
	if lesson == nil {
		return nil, ErrLessonNotFound
	}

	if lesson.Status != lessonDomain.LessonStatusLive {
		return nil, ErrLessonNotLive
	}

	now := s.clock.Now()
	lesson.Status = lessonDomain.LessonStatusCompleted
	lesson.EndsAt = &now
	lesson.UpdatedAt = now

	if err := s.lessonRepo.Update(ctx, lesson); err != nil {
		return nil, err
	}

	// Emit event
	s.events.EmitLessonEnded(ctx, events.LessonEndedPayload{
		LessonID: lesson.ID,
		CourseID: lesson.CourseID,
		EndsAt:   *lesson.EndsAt,
	})

	// Trigger progress recomputation for all students in the course
	go s.recomputeAllStudentsProgress(lesson.CourseID)

	return lesson, nil
}

// CancelLesson cancels a lesson
func (s *Service) CancelLesson(ctx context.Context, lessonID uuid.UUID) (*lessonDomain.Lesson, error) {
	lesson, err := s.lessonRepo.GetByID(ctx, lessonID)
	if err != nil {
		return nil, err
	}
	if lesson == nil {
		return nil, ErrLessonNotFound
	}

	if lesson.Status == lessonDomain.LessonStatusLive {
		return nil, ErrLessonInProgress
	}
	if lesson.Status == lessonDomain.LessonStatusCompleted {
		return nil, ErrInvalidStatus
	}

	lesson.Status = lessonDomain.LessonStatusCanceled
	lesson.UpdatedAt = s.clock.Now()

	if err := s.lessonRepo.Update(ctx, lesson); err != nil {
		return nil, err
	}

	// Emit event
	s.events.Dispatch(ctx, events.TypeLessonCanceled, lesson.ID.String(), uuid.UUID{}, events.LessonCanceledPayload{
		LessonID: lesson.ID,
		CourseID: lesson.CourseID,
		Reason:   "Canceled by teacher/assistant",
	})

	return lesson, nil
}

// RescheduleLesson reschedules a lesson to a new time
func (s *Service) RescheduleLesson(ctx context.Context, lessonID uuid.UUID, newScheduledAt time.Time) (*lessonDomain.Lesson, error) {
	lesson, err := s.lessonRepo.GetByID(ctx, lessonID)
	if err != nil {
		return nil, err
	}
	if lesson == nil {
		return nil, ErrLessonNotFound
	}

	if lesson.Status != lessonDomain.LessonStatusScheduled {
		return nil, ErrInvalidStatus
	}

	oldScheduledAt := lesson.ScheduledAt
	lesson.ScheduledAt = newScheduledAt.UTC()
	lesson.UpdatedAt = s.clock.Now()

	if err := s.lessonRepo.Update(ctx, lesson); err != nil {
		return nil, err
	}

	// Emit event
	s.events.Dispatch(ctx, events.TypeLessonRescheduled, lesson.ID.String(), uuid.UUID{}, events.LessonRescheduledPayload{
		LessonID:       lesson.ID,
		OldScheduledAt: oldScheduledAt,
		NewScheduledAt: lesson.ScheduledAt,
	})

	return lesson, nil
}

// GetUpcomingLessons returns upcoming lessons for a course
func (s *Service) GetUpcomingLessons(ctx context.Context, courseID uuid.UUID, limit int) ([]lessonDomain.Lesson, error) {
	if limit <= 0 {
		limit = 5
	}
	return s.lessonRepo.GetUpcoming(ctx, courseID, limit)
}

func (s *Service) recomputeAllStudentsProgress(courseID uuid.UUID) {
	// Normally this would be a Kafka event "lesson.completed"
	// but for now we'll do it via the progress service directly
	// Note: We need a way to get all student IDs for this course.
	// Since we don't have enrollment repo here yet, we'll leave it as a hook
	// for Phase 7 (Events).
}


// UpdateLesson updates a lesson
func (s *Service) UpdateLesson(ctx context.Context, lesson *lessonDomain.Lesson) error {
	lesson.UpdatedAt = s.clock.Now()
	return s.lessonRepo.Update(ctx, lesson)
}
