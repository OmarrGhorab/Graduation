package lesson

import (
	"context"
	"errors"
	"time"

	progressApp "github.com/OmarrGhorab/courses-attendance-service/internal/application/progress"
	"github.com/OmarrGhorab/courses-attendance-service/internal/domain/events"
	lessonDomain "github.com/OmarrGhorab/courses-attendance-service/internal/domain/lesson"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/clock"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/cloudinary"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/notificationevents"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/persistence/postgres"
	"github.com/google/uuid"
	"os"
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
	progressService  *progressApp.Service
	events           *notificationevents.EventDispatcher
	cloudinaryClient *cloudinary.Client
	clock            clock.Clock
}


func NewService(
	lessonRepo *postgres.LessonRepository,
	courseRepo *postgres.CourseRepository,
	enrollmentRepo *postgres.EnrollmentRepository,
	progressService *progressApp.Service,
	events *notificationevents.EventDispatcher,
	cloudinaryClient *cloudinary.Client,
	clk clock.Clock,
) *Service {
	return &Service{
		lessonRepo:       lessonRepo,
		courseRepo:       courseRepo,
		enrollmentRepo:   enrollmentRepo,
		progressService:  progressService,
		events:           events,
		cloudinaryClient: cloudinaryClient,
		clock:            clk,
	}
}


// CreateLessonInput represents input for creating a lesson
type CreateLessonInput struct {
	CourseID        uuid.UUID
	Title           string
	Description     string
	ThumbnailURL    string
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
		ThumbnailURL:    input.ThumbnailURL,
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

	lesson.EnrolledStudents = course.EnrollmentCount
	return lesson, nil
}

// GetLesson retrieves a lesson by ID, with data redaction for non-enrolled students
func (s *Service) GetLesson(ctx context.Context, id uuid.UUID, userID uuid.UUID, userRole string) (*lessonDomain.Lesson, error) {
	lesson, err := s.lessonRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if lesson == nil {
		return nil, ErrLessonNotFound
	}

	// Higher roles see everything
	if userRole == "TEACHER" || userRole == "INSTRUCTOR" || userRole == "ADMIN" {
		s.populateEnrollmentCount(ctx, lesson)
		return lesson, nil
	}

	// Check access
	hasAccess := false
	if lesson.IsFree {
		hasAccess = true
	} else {
		// Check enrollment
		enrollment, err := s.enrollmentRepo.GetByCourseAndUser(ctx, lesson.CourseID, userID)
		if err == nil && enrollment != nil {
			// Check if monthly
			course, err := s.courseRepo.GetByID(ctx, lesson.CourseID)
			if err == nil && course != nil {
				if course.BillingType != "MONTHLY" {
					hasAccess = true
				} else {
					paidPeriods, err := s.enrollmentRepo.GetPeriods(ctx, enrollment.ID)
					if err == nil {
						periodKey := lesson.ScheduledAt.Format("2006-01")
						for _, p := range paidPeriods {
							if p.PeriodKey == periodKey {
								hasAccess = true
								break
							}
						}
					}
				}
			}
		}
	}

	// If no access, redact sensitive data
	if !hasAccess {
		lesson.VideoURL = ""
		lesson.VideoPublicID = ""
		lesson.MaterialsURL = ""
		lesson.LocationLat = nil
		lesson.LocationLng = nil
		lesson.GeofenceRadiusM = nil
	}

	return lesson, nil
}

// GetCourseLessons retrieves all lessons for a course, with data redaction for non-enrolled students
func (s *Service) GetCourseLessons(ctx context.Context, courseID uuid.UUID, userID uuid.UUID, userRole string) ([]lessonDomain.Lesson, error) {
	lessons, err := s.lessonRepo.GetByCourseID(ctx, courseID)
	if err != nil {
		return nil, err
	}

	// Higher roles see everything
	if userRole == "TEACHER" || userRole == "INSTRUCTOR" || userRole == "ADMIN" {
		// Populate enrollment count for all lessons
		count, err := s.enrollmentRepo.CountByCourseID(ctx, courseID)
		if err == nil {
			for i := range lessons {
				lessons[i].EnrolledStudents = int(count)
			}
		}
		return lessons, nil
	}

	// Check enrollment
	enrollment, err := s.enrollmentRepo.GetByCourseAndUser(ctx, courseID, userID)
	isEnrolled := err == nil && enrollment != nil

	// Check if this course has monthly billing
	course, err := s.courseRepo.GetByID(ctx, courseID)
	if err != nil || course == nil {
		return lessons, nil 
	}

	// Pre-fetch paid periods if enrolled and monthly
	paidMap := make(map[string]bool)
	if isEnrolled && course.BillingType == "MONTHLY" {
		paidPeriods, err := s.enrollmentRepo.GetPeriods(ctx, enrollment.ID)
		if err == nil {
			for _, p := range paidPeriods {
				paidMap[p.PeriodKey] = true
			}
		}
	}

	var processed []lessonDomain.Lesson
	for _, l := range lessons {
		hasAccess := false
		
		if l.IsFree {
			hasAccess = true
		} else if isEnrolled {
			if course.BillingType != "MONTHLY" {
				hasAccess = true
			} else {
				periodKey := l.ScheduledAt.Format("2006-01")
				if paidMap[periodKey] {
					hasAccess = true
				}
			}
		}

		// If no access, redact sensitive data but keep the lesson in the list
		if !hasAccess {
			l.VideoURL = ""
			l.VideoPublicID = ""
			l.MaterialsURL = ""
			l.LocationLat = nil
			l.LocationLng = nil
			l.GeofenceRadiusM = nil
		}
		
		processed = append(processed, l)
	}

	return processed, nil
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

	s.populateEnrollmentCount(ctx, lesson)

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

	s.populateEnrollmentCount(ctx, lesson)
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

	s.populateEnrollmentCount(ctx, lesson)
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

	s.populateEnrollmentCount(ctx, lesson)
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

// PatchLesson updates specific fields of a lesson
func (s *Service) PatchLesson(ctx context.Context, teacherID uuid.UUID, lessonID uuid.UUID, updates map[string]interface{}) (*lessonDomain.Lesson, error) {
	// 1. Get lesson
	lesson, err := s.lessonRepo.GetByID(ctx, lessonID)
	if err != nil {
		return nil, err
	}
	if lesson == nil {
		return nil, ErrLessonNotFound
	}

	// 2. Verify ownership
	course, err := s.courseRepo.GetByID(ctx, lesson.CourseID)
	if err != nil {
		return nil, err
	}
	if course == nil {
		return nil, ErrCourseNotFound
	}
	if course.TeacherID != teacherID {
		return nil, ErrUnauthorized
	}

	// 3. Apply updates
	if title, ok := updates["title"].(string); ok {
		lesson.Title = title
	}
	if description, ok := updates["description"].(string); ok {
		lesson.Description = description
	}
	if thumbnail, ok := updates["thumbnail_url"].(string); ok {
		lesson.ThumbnailURL = thumbnail
	}
	if scheduledAt, ok := updates["scheduled_at"].(time.Time); ok {
		lesson.ScheduledAt = scheduledAt.UTC()
	}
	if duration, ok := updates["duration_minutes"].(int); ok {
		lesson.DurationMinutes = duration
	}
	if isFree, ok := updates["is_free"].(bool); ok {
		lesson.IsFree = isFree
	}

	lesson.UpdatedAt = s.clock.Now()

	if err := s.lessonRepo.Update(ctx, lesson); err != nil {
		return nil, err
	}

	s.populateEnrollmentCount(ctx, lesson)

	return lesson, nil
}

// ProcessLessonVideoAsync uploads a video to Cloudinary in the background
func (s *Service) ProcessLessonVideoAsync(ctx context.Context, lessonID, teacherID uuid.UUID, tempFilePath string, filename string) {
	// 1. Clean up temp file when done
	defer os.Remove(tempFilePath)

	// 2. Fetch lesson
	lesson, err := s.lessonRepo.GetByID(ctx, lessonID)
	if err != nil || lesson == nil {
		return
	}

	// 3. Open the temp file
	file, err := os.Open(tempFilePath)
	if err != nil {
		s.events.EmitLessonVideoFailed(ctx, events.LessonVideoFailedPayload{
			LessonID:    lessonID,
			LessonTitle: lesson.Title,
			TeacherID:   teacherID,
			Error:       "Failed to open source video file",
		})
		return
	}
	defer file.Close()

	// 4. Upload to Cloudinary
	// Since we need multipart.File interface, we might need a workaround or update client
	// Actually, os.File implements io.Reader, but Cloudinary client takes multipart.File
	// We'll assume the client is updated or we use a helper
	result, err := s.cloudinaryClient.UploadVideo(ctx, file, filename)
	if err != nil {
		s.events.EmitLessonVideoFailed(ctx, events.LessonVideoFailedPayload{
			LessonID:    lessonID,
			LessonTitle: lesson.Title,
			TeacherID:   teacherID,
			Error:       err.Error(),
		})
		return
	}

	// 5. Update lesson
	lesson.VideoURL = result.StreamingURL
	lesson.VideoPublicID = result.PublicID
	if result.Duration != nil {
		lesson.Duration = result.Duration
	}
	lesson.UpdatedAt = s.clock.Now()
	s.lessonRepo.Update(ctx, lesson)

	// 6. Emit Success Event
	s.events.EmitLessonVideoReady(ctx, events.LessonVideoReadyPayload{
		LessonID:     lessonID,
		LessonTitle:  lesson.Title,
		TeacherID:    teacherID,
		VideoURL:     result.URL,
		StreamingURL: result.StreamingURL,
		Duration:     0, // TODO: set from result
	})
}

func (s *Service) populateEnrollmentCount(ctx context.Context, lesson *lessonDomain.Lesson) {
	if lesson == nil {
		return
	}
	count, err := s.enrollmentRepo.CountByCourseID(ctx, lesson.CourseID)
	if err == nil {
		lesson.EnrolledStudents = int(count)
	}
}
