package dto

import (
	"time"

	"github.com/google/uuid"
)

// StudentAnalyticsResponse represents analytics for a student in a course
type StudentAnalyticsResponse struct {
	StudentID        uuid.UUID              `json:"studentId"`
	StudentName      string                 `json:"studentName"`
	StudentProfileImg string                `json:"studentProfileImg"`
	CourseID         uuid.UUID              `json:"courseId"`
	CourseName       string                 `json:"courseName"`
	
	// Attendance metrics
	AttendanceRate   float64                `json:"attendanceRate"`   // Percentage
	AttendanceChange float64                `json:"attendanceChange"` // Change from previous period
	
	// Completion metrics
	CompletionRate   float64                `json:"completionRate"`   // Percentage
	CompletedLessons int                    `json:"completedLessons"`
	TotalLessons     int                    `json:"totalLessons"`
	
	// Weekly attendance (hours per day)
	WeeklyAttendance []DailyAttendance      `json:"weeklyAttendance"`
	
	// Ranking
	Rank             int                    `json:"rank"`
	TotalStudents    int                    `json:"totalStudents"`
	Points           int                    `json:"points"`
	
	// Recent activity
	RecentActivity   []RecentActivityItem   `json:"recentActivity"`
}

// DailyAttendance represents attendance hours for a specific day
type DailyAttendance struct {
	Day   string  `json:"day"`   // Mon, Tue, Wed, etc.
	Hours float64 `json:"hours"` // Hours attended
}

// RecentActivityItem represents a recent lesson attendance
type RecentActivityItem struct {
	LessonID     uuid.UUID  `json:"lessonId"`
	LessonTitle  string     `json:"lessonTitle"`
	Status       string     `json:"status"`       // PRESENT, LATE, ABSENT, EXCUSED
	ScheduledAt  time.Time  `json:"scheduledAt"`
	ScannedAt    *time.Time `json:"scannedAt,omitempty"`
	DurationMins int        `json:"durationMins"`
}
