package dto

import (
	"time"

	"github.com/OmarrGhorab/courses-attendance-service/internal/domain/progress"
	"github.com/google/uuid"
)

type ProgressResponse struct {
	ID               uuid.UUID `json:"id"`
	CourseID         uuid.UUID `json:"courseId"`
	StudentID        uuid.UUID `json:"studentId"`
	TotalLessons     int       `json:"totalLessons"`
	CompletedLessons int       `json:"completedLessons"`
	PresentCount     int       `json:"presentCount"`
	LateCount        int       `json:"lateCount"`
	AbsentCount      int       `json:"absentCount"`
	ExcusedCount     int       `json:"excusedCount"`
	CompletionRatio  float64   `json:"completionRatio"`
	AttendanceRatio  float64   `json:"attendanceRatio"`
	OverallProgress  float64   `json:"overallProgress"`
	CalculatedAt     time.Time `json:"calculatedAt"`
}

func ToProgressResponse(p *progress.ProgressSnapshot) ProgressResponse {
	return ProgressResponse{
		ID:               p.ID,
		CourseID:         p.CourseID,
		StudentID:        p.StudentID,
		TotalLessons:     p.TotalLessons,
		CompletedLessons: p.CompletedLessons,
		PresentCount:     p.PresentCount,
		LateCount:        p.LateCount,
		AbsentCount:      p.AbsentCount,
		ExcusedCount:     p.ExcusedCount,
		CompletionRatio:  p.CompletionRatio,
		AttendanceRatio:  p.AttendanceRatio,
		OverallProgress:  p.OverallProgress,
		CalculatedAt:     p.CalculatedAt,
	}
}

type CalendarEventResponse struct {
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
