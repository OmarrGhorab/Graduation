package progress

import (
	"context"
	"fmt"

	attendanceDomain "github.com/OmarrGhorab/courses-attendance-service/internal/domain/attendance"
	"github.com/OmarrGhorab/courses-attendance-service/internal/domain/events"
	lessonDomain "github.com/OmarrGhorab/courses-attendance-service/internal/domain/lesson"
	"github.com/OmarrGhorab/courses-attendance-service/internal/domain/progress"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/clock"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/notificationevents"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/persistence/postgres"
	"github.com/google/uuid"
)

// Service handles student progress calculation and snapshots
type Service struct {
	progressRepo *postgres.ProgressSnapshotRepository
	recordRepo   *postgres.AttendanceRecordRepository
	courseRepo   *postgres.CourseRepository
	lessonRepo   *postgres.LessonRepository
	events       *notificationevents.EventDispatcher
	clock        clock.Clock
}

func NewService(
	progressRepo *postgres.ProgressSnapshotRepository,
	recordRepo *postgres.AttendanceRecordRepository,
	courseRepo *postgres.CourseRepository,
	lessonRepo *postgres.LessonRepository,
	events *notificationevents.EventDispatcher,
	clk clock.Clock,
) *Service {
	return &Service{
		progressRepo: progressRepo,
		recordRepo:   recordRepo,
		courseRepo:   courseRepo,
		lessonRepo:   lessonRepo,
		events:       events,
		clock:        clk,
	}
}

// RecomputeProgress calculates and saves a new progress snapshot for a student in a course
func (s *Service) RecomputeProgress(ctx context.Context, courseID, studentID uuid.UUID) (*progress.ProgressSnapshot, error) {
	// 1. Get course to find attendance weight
	course, err := s.courseRepo.GetByID(ctx, courseID)
	if err != nil {
		return nil, fmt.Errorf("failed to get course: %w", err)
	}
	if course == nil {
		return nil, fmt.Errorf("course not found")
	}

	// 2. Get all lessons for the course
	lessons, err := s.lessonRepo.GetByCourseID(ctx, courseID)
	if err != nil {
		return nil, fmt.Errorf("failed to get lessons: %w", err)
	}

	counts, err := s.recordRepo.CountByStudentAndCourse(ctx, studentID, courseID)
	if err != nil {
		return nil, fmt.Errorf("failed to count attendance records: %w", err)
	}

	// 4. Aggregate metrics
	snapshot := &progress.ProgressSnapshot{
		CourseID:     courseID,
		StudentID:    studentID,
		TotalLessons: course.TotalLessons,
		CalculatedAt: s.clock.Now(),
	}

	// Check if snapshot already exists
	existing, err := s.progressRepo.GetByCourseAndStudent(ctx, courseID, studentID)
	if err == nil && existing != nil {
		snapshot.ID = existing.ID
	} else {
		snapshot.ID = uuid.New()
	}

	// Count completed lessons (lessons that are COMPLETED)
	completedCount := 0
	for _, l := range lessons {
		if l.Status == lessonDomain.LessonStatusCompleted {
			completedCount++
		}
	}
	snapshot.CompletedLessons = completedCount

	// Use counts from repo
	snapshot.PresentCount = counts[attendanceDomain.AttendanceStatusPresent]
	snapshot.LateCount = counts[attendanceDomain.AttendanceStatusLate]
	snapshot.AbsentCount = counts[attendanceDomain.AttendanceStatusAbsent]
	snapshot.ExcusedCount = counts[attendanceDomain.AttendanceStatusExcused]

	// 5. Calculate ratios and overall progress
	snapshot.Calculate(course.AttendanceWeight)

	// 6. Save snapshot
	if err := s.progressRepo.Upsert(ctx, snapshot); err != nil {
		return nil, fmt.Errorf("failed to save progress snapshot: %w", err)
	}

	// Emit event
	s.events.EmitProgressUpdated(ctx, events.ProgressUpdatedPayload{
		CourseID:        courseID,
		StudentID:       studentID,
		OverallProgress: snapshot.OverallProgress,
	})

	return snapshot, nil
}

// GetStudentProgress returns the latest progress for a student in a course
func (s *Service) GetStudentProgress(ctx context.Context, courseID, studentID uuid.UUID) (*progress.ProgressSnapshot, error) {
	return s.progressRepo.GetByCourseAndStudent(ctx, courseID, studentID)
}

// GetCourseProgress returns progress for all students in a course
func (s *Service) GetCourseProgress(ctx context.Context, courseID uuid.UUID) ([]progress.ProgressSnapshot, error) {
	return s.progressRepo.GetByCourseID(ctx, courseID)
}
