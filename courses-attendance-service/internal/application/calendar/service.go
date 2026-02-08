package calendar

import (
	"context"
	"time"

	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/persistence/postgres"
	"github.com/google/uuid"
)

// CalendarEvent represents a simplified lesson for calendar display
type CalendarEvent struct {
	ID           uuid.UUID `json:"id"`
	Title        string    `json:"title"`
	CourseID     uuid.UUID `json:"courseId"`
	CourseTitle  string    `json:"courseTitle"`
	StartTime    time.Time `json:"startTime"`
	EndTime      time.Time `json:"endTime"`
	Status       string    `json:"status"`
	Location     string    `json:"location"`
	LessonNumber int       `json:"lessonNumber"`
}

// Service handles calendar-related queries
type Service struct {
	lessonRepo     *postgres.LessonRepository
	courseRepo     *postgres.CourseRepository
	enrollmentRepo *postgres.EnrollmentRepository
}

func NewService(
	lessonRepo *postgres.LessonRepository,
	courseRepo *postgres.CourseRepository,
	enrollmentRepo *postgres.EnrollmentRepository,
) *Service {
	return &Service{
		lessonRepo:     lessonRepo,
		courseRepo:     courseRepo,
		enrollmentRepo: enrollmentRepo,
	}
}

// GetStudentCalendar returns all upcoming lessons for a student
func (s *Service) GetStudentCalendar(ctx context.Context, studentID uuid.UUID, start, end time.Time) ([]CalendarEvent, error) {
	// 1. Get student enrollments
	enrollments, err := s.enrollmentRepo.GetByUserID(ctx, studentID)
	if err != nil {
		return nil, err
	}

	if len(enrollments) == 0 {
		return []CalendarEvent{}, nil
	}

	courseIDs := make([]uuid.UUID, len(enrollments))
	courseMap := make(map[uuid.UUID]string)
	for i, e := range enrollments {
		courseIDs[i] = e.CourseID
	}

	// 2. Get courses for titles
	courses, err := s.courseRepo.GetByIDs(ctx, courseIDs)
	if err != nil {
		return nil, err
	}
	for _, c := range courses {
		courseMap[c.ID] = c.Title
	}

	// 3. Get lessons for these courses in time range
	// We need a repository method for this
	lessons, err := s.lessonRepo.GetByCoursesAndTimeRange(ctx, courseIDs, start, end)
	if err != nil {
		return nil, err
	}

	events := make([]CalendarEvent, len(lessons))
	for i, l := range lessons {
		endTime := l.ScheduledAt.Add(time.Duration(l.DurationMinutes) * time.Minute)

		events[i] = CalendarEvent{
			ID:           l.ID,
			Title:        l.Title,
			CourseID:     l.CourseID,
			CourseTitle:  courseMap[l.CourseID],
			StartTime:    l.ScheduledAt,
			EndTime:      endTime,
			Status:       string(l.Status),
			Location:     l.LocationName,
			LessonNumber: l.LessonNumber,
		}
	}

	return events, nil
}

// GetTeacherCalendar returns all scheduled lessons for a teacher
func (s *Service) GetTeacherCalendar(ctx context.Context, teacherID uuid.UUID, start, end time.Time) ([]CalendarEvent, error) {
	// 1. Get teacher courses
	courses, err := s.courseRepo.GetByTeacherID(ctx, teacherID)
	if err != nil {
		return nil, err
	}

	if len(courses) == 0 {
		return []CalendarEvent{}, nil
	}

	courseIDs := make([]uuid.UUID, len(courses))
	courseMap := make(map[uuid.UUID]string)
	for i, c := range courses {
		courseIDs[i] = c.ID
		courseMap[c.ID] = c.Title
	}

	// 2. Get lessons
	lessons, err := s.lessonRepo.GetByCoursesAndTimeRange(ctx, courseIDs, start, end)
	if err != nil {
		return nil, err
	}

	events := make([]CalendarEvent, len(lessons))
	for i, l := range lessons {
		endTime := l.ScheduledAt.Add(time.Duration(l.DurationMinutes) * time.Minute)

		events[i] = CalendarEvent{
			ID:           l.ID,
			Title:        l.Title,
			CourseID:     l.CourseID,
			CourseTitle:  courseMap[l.CourseID],
			StartTime:    l.ScheduledAt,
			EndTime:      endTime,
			Status:       string(l.Status),
			Location:     l.LocationName,
			LessonNumber: l.LessonNumber,
		}
	}

	return events, nil
}
