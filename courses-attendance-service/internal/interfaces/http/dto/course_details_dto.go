package dto

import (
	"time"

	"github.com/google/uuid"
)

// CourseDetailsResponse combines course info, progress, and lessons in one response
type CourseDetailsResponse struct {
	Course   CourseInfo        `json:"course"`
	Progress *ProgressInfo     `json:"progress,omitempty"`
	Teacher  *TeacherInfo      `json:"teacher,omitempty"`
	Lessons  []LessonInfo      `json:"lessons"`
}

// CourseInfo contains basic course information
type CourseInfo struct {
	ID                      uuid.UUID `json:"id"`
	Title                   string    `json:"title"`
	Code                    string    `json:"code,omitempty"` // Can be derived from subject + course number
	Level                   string    `json:"level,omitempty"`
	Description             string    `json:"description"`
	SubjectID               uuid.UUID `json:"subjectId"`
	SubjectName             string    `json:"subjectName"`
	CourseImage             string    `json:"courseImage,omitempty"`
	DeliveryType            string    `json:"deliveryType"`
	LocationName            string    `json:"locationName,omitempty"`
	TotalLessons            int       `json:"totalLessons"`
	AttendanceWindowMinutes int       `json:"attendanceWindowMinutes"`
	Price                   float64   `json:"price"`
	Currency                string    `json:"currency"`
	IsPaid                  bool      `json:"isPaid"`
	BillingType             string    `json:"billingType"`
	Status                  string    `json:"status"`
	AttendanceWeight        float64   `json:"attendanceWeight"`
	EnrollmentCount         int       `json:"enrollmentCount"` // Total students enrolled
	Assistants              []AssistantInfo `json:"assistants,omitempty"` // Course assistants
}

// ProgressInfo contains student progress information
type ProgressInfo struct {
	AttendancePercentage float64 `json:"attendancePercentage"`
	ClassesAttended      int     `json:"classesAttended"`
	TotalClasses         int     `json:"totalClasses"`
	OverallGrade         float64 `json:"overallGrade"`
	Status               string  `json:"status"` // "Good Standing", "At Risk", etc.
	TargetPercentage     float64 `json:"targetPercentage"`
	PresentCount         int     `json:"presentCount"`
	LateCount            int     `json:"lateCount"`
	AbsentCount          int     `json:"absentCount"`
	ExcusedCount         int     `json:"excusedCount"`
	LastUpdated          time.Time `json:"lastUpdated"`
}

// TeacherInfo contains teacher information
type TeacherInfo struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Title       string    `json:"title,omitempty"`
	ProfileImg  string    `json:"profileImg,omitempty"`
	Department  string    `json:"department,omitempty"`
}

// LessonInfo contains lesson information for the syllabus
type LessonInfo struct {
	ID                uuid.UUID  `json:"id"`
	Title             string     `json:"title"`
	Description       string     `json:"description,omitempty"`
	LessonNumber      int        `json:"lessonNumber"`
	Status            string     `json:"status"` // LIVE, COMPLETED, UPCOMING, CANCELED
	ScheduledAt       time.Time  `json:"scheduledAt"`
	StartsAt          *time.Time `json:"startsAt,omitempty"`
	EndsAt            *time.Time `json:"endsAt,omitempty"`
	DurationMinutes   int        `json:"durationMinutes"`
	DeliveryType      string     `json:"deliveryType"`
	LocationName      string     `json:"locationName,omitempty"`
	LocationLat       *float64   `json:"locationLat,omitempty"`
	LocationLng       *float64   `json:"locationLng,omitempty"`
	CanMarkAttendance bool       `json:"canMarkAttendance"` // True if lesson is LIVE
	AttendanceStatus  *string    `json:"attendanceStatus,omitempty"` // PRESENT, LATE, ABSENT, EXCUSED (null if no record)
	AttendeeCount     *int       `json:"attendeeCount,omitempty"` // Number of students who attended
	AbsenceRequestStatus *string `json:"absenceRequestStatus,omitempty"` // PENDING, APPROVED, REJECTED (null if no request)
}

// AssistantInfo contains assistant information for a course
type AssistantInfo struct {
	ID                uuid.UUID `json:"id"`
	AssistantID       uuid.UUID `json:"assistantId"`
	AssistantName     string    `json:"assistantName,omitempty"`
	AssistantProfileImg string  `json:"assistantProfileImg,omitempty"`
	CanStartLesson    bool      `json:"canStartLesson"`
	CanEndLesson      bool      `json:"canEndLesson"`
	CanViewAttendance bool      `json:"canViewAttendance"`
	CanEditAttendance bool      `json:"canEditAttendance"`
	AddedAt           time.Time `json:"addedAt"`
}
