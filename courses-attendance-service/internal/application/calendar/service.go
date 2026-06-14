package calendar

import (
	"context"
	"time"

	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/persistence/postgres"
	"github.com/google/uuid"
)

// CalendarEvent represents a simplified lesson for calendar display
type CalendarEvent struct {
	ID                 uuid.UUID  `json:"id"`
	Title              string     `json:"title"`
	CourseID           uuid.UUID  `json:"courseId"`
	CourseTitle        string     `json:"courseTitle"`
	StartTime          time.Time  `json:"startTime"`
	EndTime            time.Time  `json:"endTime"`
	Status             string     `json:"status"`
	Location           string     `json:"location"`
	LessonNumber       int        `json:"lessonNumber"`
	AttendanceStatus   *string    `json:"attendanceStatus"`
	CanMarkAttendance  bool       `json:"canMarkAttendance"`
	ChildID            *uuid.UUID `json:"childId,omitempty"`
	ChildName          string     `json:"childName,omitempty"`
}

// Service handles calendar-related queries
type Service struct {
	lessonRepo     *postgres.LessonRepository
	courseRepo     *postgres.CourseRepository
	enrollmentRepo *postgres.EnrollmentRepository
	attendanceRepo *postgres.AttendanceRecordRepository
}

func NewService(
	lessonRepo *postgres.LessonRepository,
	courseRepo *postgres.CourseRepository,
	enrollmentRepo *postgres.EnrollmentRepository,
	attendanceRepo *postgres.AttendanceRecordRepository,
) *Service {
	return &Service{
		lessonRepo:     lessonRepo,
		courseRepo:     courseRepo,
		enrollmentRepo: enrollmentRepo,
		attendanceRepo: attendanceRepo,
	}
}

// CalendarFilter represents the filter criteria for calendar events
type CalendarFilter struct {
	Start       time.Time
	End         time.Time
	SubjectID   *uuid.UUID
	SubjectName string
	Statuses    []string // e.g., ["SCHEDULED", "COMPLETED"]
	Page        int
	Limit       int
}

// GetStudentCalendar returns all upcoming lessons for a student with attendance status
func (s *Service) GetStudentCalendar(ctx context.Context, studentID uuid.UUID, filter CalendarFilter) ([]CalendarEvent, int64, error) {
	enrollments, err := s.enrollmentRepo.GetByUserID(ctx, studentID)
	if err != nil {
		return nil, 0, err
	}
	if len(enrollments) == 0 {
		return []CalendarEvent{}, 0, nil
	}

	courseIDs := make([]uuid.UUID, len(enrollments))
	for i, e := range enrollments {
		courseIDs[i] = e.CourseID
	}

	offset := (filter.Page - 1) * filter.Limit
	if offset < 0 {
		offset = 0
	}

	lessons, total, err := s.lessonRepo.GetFilteredLessons(ctx, courseIDs, filter.SubjectID, filter.SubjectName, filter.Statuses, filter.Start, filter.End, filter.Limit, offset)
	if err != nil {
		return nil, 0, err
	}
	if len(lessons) == 0 {
		return []CalendarEvent{}, total, nil
	}

	neededCourseIDs := make(map[uuid.UUID]bool)
	lessonIDs := make([]uuid.UUID, len(lessons))
	for i, l := range lessons {
		neededCourseIDs[l.CourseID] = true
		lessonIDs[i] = l.ID
	}

	uniqueCourseIDs := make([]uuid.UUID, 0, len(neededCourseIDs))
	for cid := range neededCourseIDs {
		uniqueCourseIDs = append(uniqueCourseIDs, cid)
	}

	courses, err := s.courseRepo.GetByIDs(ctx, uniqueCourseIDs)
	if err != nil {
		return nil, 0, err
	}
	courseMap := make(map[uuid.UUID]string)
	for _, c := range courses {
		courseMap[c.ID] = c.Title
	}

	// Batch-fetch attendance records for this student across all lesson IDs
	attendanceMap := make(map[uuid.UUID]string)
	records, err := s.attendanceRepo.GetByStudentAndLessons(ctx, studentID, lessonIDs)
	if err == nil {
		for _, rec := range records {
			attendanceMap[rec.LessonID] = string(rec.Status)
		}
	}

	now := time.Now()
	events := make([]CalendarEvent, len(lessons))
	for i, l := range lessons {
		endTime := l.ScheduledAt.Add(time.Duration(l.DurationMinutes) * time.Minute)

		var attendanceStatus *string
		if status, ok := attendanceMap[l.ID]; ok {
			attendanceStatus = &status
		}

		canMark := string(l.Status) == "LIVE" && attendanceStatus == nil && now.Before(endTime)

		events[i] = CalendarEvent{
			ID:                l.ID,
			Title:             l.Title,
			CourseID:          l.CourseID,
			CourseTitle:       courseMap[l.CourseID],
			StartTime:         l.ScheduledAt,
			EndTime:           endTime,
			Status:            string(l.Status),
			Location:          l.LocationName,
			LessonNumber:      l.LessonNumber,
			AttendanceStatus:  attendanceStatus,
			CanMarkAttendance: canMark,
		}
	}

	return events, total, nil
}

// GetParentCalendar returns merged calendar events for all children of a parent
func (s *Service) GetParentCalendar(ctx context.Context, childIDs []uuid.UUID, childNameMap map[uuid.UUID]string, filter CalendarFilter) ([]CalendarEvent, int64, error) {
	if len(childIDs) == 0 {
		return []CalendarEvent{}, 0, nil
	}

	var allEvents []CalendarEvent
	var maxTotal int64

	for _, childID := range childIDs {
		events, total, err := s.GetStudentCalendar(ctx, childID, filter)
		if err != nil {
			continue
		}
		childName := childNameMap[childID]
		for i := range events {
			cid := childID
			events[i].ChildID = &cid
			events[i].ChildName = childName
		}
		allEvents = append(allEvents, events...)
		if total > maxTotal {
			maxTotal = total
		}
	}

	return allEvents, maxTotal, nil
}

// GetTeacherCalendar returns all scheduled lessons for a teacher
func (s *Service) GetTeacherCalendar(ctx context.Context, teacherID uuid.UUID, filter CalendarFilter) ([]CalendarEvent, int64, error) {
	// 1. Get teacher courses
	courses, err := s.courseRepo.GetByTeacherID(ctx, teacherID)
	if err != nil {
		return nil, 0, err
	}

	if len(courses) == 0 {
		return []CalendarEvent{}, 0, nil
	}

	courseIDs := make([]uuid.UUID, len(courses))
	courseMap := make(map[uuid.UUID]string)
	for i, c := range courses {
		courseIDs[i] = c.ID
		courseMap[c.ID] = c.Title
	}

	// 2. Get filtered lessons
	offset := 0
	if filter.Page > 1 && filter.Limit > 0 {
		offset = (filter.Page - 1) * filter.Limit
	}

	lessons, total, err := s.lessonRepo.GetFilteredLessons(ctx, courseIDs, filter.SubjectID, filter.SubjectName, filter.Statuses, filter.Start, filter.End, filter.Limit, offset)
	if err != nil {
		return nil, 0, err
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

	return events, total, nil
}
