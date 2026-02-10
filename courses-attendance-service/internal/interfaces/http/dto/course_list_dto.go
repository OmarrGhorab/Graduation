package dto

import (
	"time"

	"github.com/google/uuid"
)

// CourseListResponse is an enhanced course response for list views with teacher info
type CourseListResponse struct {
	ID                      uuid.UUID    `json:"id"`
	Title                   string       `json:"title"`
	Description             string       `json:"description"`
	SubjectID               uuid.UUID    `json:"subjectId"`
	SubjectName             string       `json:"subjectName,omitempty"`
	TeacherID               uuid.UUID    `json:"teacherId"`
	TeacherName             string       `json:"teacherName,omitempty"`
	TeacherProfileImg       string       `json:"teacherProfileImg,omitempty"`
	TeacherRating           float64      `json:"teacherRating,omitempty"`
	CourseImage             string       `json:"courseImage,omitempty"`
	CourseRating            float64      `json:"courseRating,omitempty"`
	TotalRatings            int          `json:"totalRatings,omitempty"`
	EnrolledStudents        int          `json:"enrolledStudents,omitempty"`
	DeliveryType            string       `json:"deliveryType"`
	LocationName            string       `json:"locationName,omitempty"`
	LocationLat             *float64     `json:"locationLat,omitempty"`
	LocationLng             *float64     `json:"locationLng,omitempty"`
	GeofenceRadiusM         int          `json:"geofenceRadiusM"`
	TotalLessons            int          `json:"totalLessons"`
	AttendanceWindowMinutes int          `json:"attendanceWindowMinutes"`
	Price                   float64      `json:"price"`
	Currency                string       `json:"currency"`
	IsPaid                  bool         `json:"isPaid"`
	BillingType             string       `json:"billingType"`
	Status                  string       `json:"status"`
	AttendanceWeight        float64      `json:"attendanceWeight"`
	CreatedAt               time.Time    `json:"createdAt"`
	UpdatedAt               time.Time    `json:"updatedAt"`
}
