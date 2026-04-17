package dto

import (
	"time"

	"github.com/OmarrGhorab/courses-attendance-service/internal/domain/course"
	lessonDomain "github.com/OmarrGhorab/courses-attendance-service/internal/domain/lesson"
	"github.com/google/uuid"
)

// ========== Course DTOs ==========

type CreateCourseRequest struct {
	Title                   string   `json:"title" validate:"required,min=3,max=255"`
	Description             string   `json:"description"`
	SubjectID               string   `json:"subjectId" validate:"required,uuid"`
	CourseImage             string   `json:"courseImage"`
	DeliveryType            string   `json:"deliveryType" validate:"required,oneof=ONLINE OFFLINE"`
	LocationName            string   `json:"locationName"`
	LocationLat             *float64 `json:"locationLat"`
	LocationLng             *float64 `json:"locationLng"`
	GeofenceRadiusM         int      `json:"geofenceRadiusM"`
	TotalLessons            int      `json:"totalLessons"`
	AttendanceWindowMinutes int      `json:"attendanceWindowMinutes"`
	Price                   float64  `json:"price"`
	Currency                string   `json:"currency"`
	IsPaid                  bool     `json:"isPaid"`
	BillingType             string   `json:"billingType" validate:"omitempty,oneof=ONE_TIME MONTHLY"`
	FreeTrialLessons        int      `json:"freeTrialLessons"` // Number of free lessons for trial
	AttendanceWeight        float64  `json:"attendanceWeight" validate:"omitempty,min=0,max=1"`
	PreviewVideoURL         string   `json:"previewVideoUrl"`
	PreviewVideoPublicID    string   `json:"previewVideoPublicId"`
}

type UpdateCourseRequest struct {
	Title                   *string  `json:"title" validate:"omitempty,min=3,max=255"`
	Description             *string  `json:"description"`
	CourseImage             *string  `json:"courseImage"`
	LocationName            *string  `json:"locationName"`
	LocationLat             *float64 `json:"locationLat"`
	LocationLng             *float64 `json:"locationLng"`
	GeofenceRadiusM         *int     `json:"geofenceRadiusM"`
	AttendanceWindowMinutes *int     `json:"attendanceWindowMinutes"`
	Price                   *float64 `json:"price"`
	BillingType             *string  `json:"billingType" validate:"omitempty,oneof=ONE_TIME MONTHLY"`
	FreeTrialLessons        *int     `json:"freeTrialLessons"` // Number of free lessons for trial
	Status                  *string  `json:"status" validate:"omitempty,oneof=ACTIVE PAUSED ARCHIVED"`
	PreviewVideoURL         *string  `json:"previewVideoUrl"`
	PreviewVideoPublicID    *string  `json:"previewVideoPublicId"`
}

type CourseResponse struct {
	ID                      uuid.UUID `json:"id"`
	Title                   string    `json:"title"`
	Description             string    `json:"description"`
	SubjectID               uuid.UUID `json:"subjectId"`
	SubjectName             string    `json:"subjectName,omitempty"`
	TeacherID               uuid.UUID `json:"teacherId"`
	CourseImage             string    `json:"courseImage,omitempty"`
	DeliveryType            string    `json:"deliveryType"`
	LocationName            string    `json:"locationName,omitempty"`
	LocationLat             *float64  `json:"locationLat,omitempty"`
	LocationLng             *float64  `json:"locationLng,omitempty"`
	GeofenceRadiusM         int       `json:"geofenceRadiusM"`
	TotalLessons            int       `json:"totalLessons"`
	AttendanceWindowMinutes int       `json:"attendanceWindowMinutes"`
	Price                   float64   `json:"price"`
	Currency                string    `json:"currency"`
	IsPaid                  bool      `json:"isPaid"`
	BillingType             string    `json:"billingType"`
	Status                  string    `json:"status"`
	AttendanceWeight        float64   `json:"attendanceWeight"`
	EnrollmentCount         int       `json:"enrollmentCount"`
	TeacherAuthority        int       `json:"teacherAuthority,omitempty"`
	TeacherName             string    `json:"teacherName,omitempty"`
	TeacherProfileImg       string    `json:"teacherProfileImg,omitempty"`
	PreviewVideoURL         string    `json:"previewVideoUrl,omitempty"`
	PreviewVideoPublicID    string    `json:"previewVideoPublicId,omitempty"`
	CreatedAt               time.Time `json:"createdAt"`
	UpdatedAt               time.Time `json:"updatedAt"`
}

func ToCourseResponse(c *course.Course) CourseResponse {
	resp := CourseResponse{
		ID:                      c.ID,
		Title:                   c.Title,
		Description:             c.Description,
		SubjectID:               c.SubjectID,
		TeacherID:               c.TeacherID,
		CourseImage:             c.CourseImage,
		DeliveryType:            string(c.DeliveryType),
		LocationName:            c.LocationName,
		LocationLat:             c.LocationLat,
		LocationLng:             c.LocationLng,
		GeofenceRadiusM:         c.GeofenceRadiusM,
		TotalLessons:            c.TotalLessons,
		AttendanceWindowMinutes: c.AttendanceWindowMinutes,
		Price:                   c.Price,
		Currency:                c.Currency,
		IsPaid:                  c.IsPaid,
		BillingType:             string(c.BillingType),
		Status:                  string(c.Status),
		AttendanceWeight:        c.AttendanceWeight,
		EnrollmentCount:         c.EnrollmentCount, // Real count from the entity
		TeacherProfileImg:       "", // Will be populated by handler
		PreviewVideoURL:         c.PreviewVideoURL,
		PreviewVideoPublicID:    c.PreviewVideoPublicID,
		CreatedAt:               c.CreatedAt,
		UpdatedAt:               c.UpdatedAt,
	}
	if c.Subject != nil {
		resp.SubjectName = c.Subject.Name
	}
	return resp
}

func ToCourseListResponse(courses []course.Course) []CourseResponse {
	res := make([]CourseResponse, len(courses))
	for i, c := range courses {
		res[i] = ToCourseResponse(&c)
	}
	return res
}

// ========== Subject DTOs ==========

type SubjectResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Icon        string    `json:"icon"`
}

func ToSubjectResponse(s *course.Subject) SubjectResponse {
	return SubjectResponse{
		ID:          s.ID,
		Name:        s.Name,
		Description: s.Description,
		Icon:        s.Icon,
	}
}

// ========== Enrollment DTOs ==========

type EnrollRequest struct {
	StudentID string `json:"studentId" validate:"required,uuid"`
}

type EnrollmentResponse struct {
	ID         uuid.UUID  `json:"id"`
	CourseID   uuid.UUID  `json:"courseId"`
	UserID     uuid.UUID  `json:"userId"`
	IsActive   bool       `json:"isActive"`
	IsPaid     bool       `json:"isPaid"`
	PaidAt     *time.Time `json:"paidAt,omitempty"`
	EnrolledAt time.Time  `json:"enrolledAt"`
}

func ToEnrollmentResponse(e *course.Enrollment) EnrollmentResponse {
	return EnrollmentResponse{
		ID:         e.ID,
		CourseID:   e.CourseID,
		UserID:     e.UserID,
		IsActive:   e.IsActive,
		IsPaid:     e.IsPaid,
		PaidAt:     e.PaidAt,
		EnrolledAt: e.EnrolledAt,
	}
}

// ========== Assistant DTOs ==========

type AddAssistantRequest struct {
	AssistantID string `json:"assistantId" validate:"required,uuid"`
}

type AssistantResponse struct {
	ID                uuid.UUID `json:"id"`
	CourseID          uuid.UUID `json:"courseId"`
	AssistantID       uuid.UUID `json:"assistantId"`
	CanStartLesson    bool      `json:"canStartLesson"`
	CanEndLesson      bool      `json:"canEndLesson"`
	CanViewAttendance bool      `json:"canViewAttendance"`
	CanEditAttendance bool      `json:"canEditAttendance"`
	CreatedAt         time.Time `json:"createdAt"`
}

func ToAssistantResponse(a *course.CourseAssistant) AssistantResponse {
	return AssistantResponse{
		ID:                a.ID,
		CourseID:          a.CourseID,
		AssistantID:       a.AssistantID,
		CanStartLesson:    a.CanStartLesson,
		CanEndLesson:      a.CanEndLesson,
		CanViewAttendance: a.CanViewAttendance,
		CanEditAttendance: a.CanEditAttendance,
		CreatedAt:         a.CreatedAt,
	}
}

// ========== Lesson DTOs ==========

type CreateLessonRequest struct {
	CourseID        string    `json:"courseId" validate:"required,uuid"`
	Title           string    `json:"title" validate:"required,min=3,max=255"`
	Description     string    `json:"description"`
	ThumbnailURL    string    `json:"thumbnailUrl"`
	ScheduledAt     time.Time `json:"scheduledAt" validate:"required"`
	DurationMinutes int       `json:"durationMinutes"`
	DeliveryType    string    `json:"deliveryType" validate:"required,oneof=ONLINE OFFLINE"`
	IsFree          bool      `json:"isFree"` // True if this lesson is free (for trial)
	
	// Online lesson materials (for ONLINE delivery type)
	VideoURL      string `json:"videoUrl"`      // Cloudinary video URL
	VideoPublicID string `json:"videoPublicId"` // Cloudinary public ID
	MaterialsURL  string `json:"materialsUrl"`  // Additional materials URL
	Duration      *int   `json:"duration"`      // Video duration in seconds
	
	// Location (for OFFLINE delivery type)
	LocationName    string   `json:"locationName"`
	LocationLat     *float64 `json:"locationLat"`
	LocationLng     *float64 `json:"locationLng"`
	GeofenceRadiusM *int     `json:"geofenceRadiusM"`
}

type RescheduleLessonRequest struct {
	ScheduledAt time.Time `json:"scheduledAt" validate:"required"`
}

type LessonResponse struct {
	ID              uuid.UUID  `json:"id"`
	CourseID        uuid.UUID  `json:"courseId"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	ThumbnailURL    string     `json:"thumbnailUrl"`
	LessonNumber    int        `json:"lessonNumber"`
	ScheduledAt     time.Time  `json:"scheduledAt"`
	StartsAt        *time.Time `json:"startsAt,omitempty"`
	EndsAt          *time.Time `json:"endsAt,omitempty"`
	DurationMinutes int        `json:"durationMinutes"`
	Status          string     `json:"status"`
	DeliveryType    string     `json:"deliveryType"`
	LocationName    string     `json:"locationName,omitempty"`
	LocationLat     *float64   `json:"locationLat,omitempty"`
	LocationLng     *float64   `json:"locationLng,omitempty"`
	GeofenceRadiusM *int       `json:"geofenceRadiusM,omitempty"`
	VideoURL        string     `json:"videoUrl,omitempty"`
	MaterialsURL    string     `json:"materialsUrl,omitempty"`
	Duration        *int       `json:"duration,omitempty"` // in seconds
	IsFree          bool       `json:"isFree"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

func ToLessonResponse(l *lessonDomain.Lesson) LessonResponse {
	return LessonResponse{
		ID:              l.ID,
		CourseID:        l.CourseID,
		Title:           l.Title,
		Description:     l.Description,
		ThumbnailURL:    l.ThumbnailURL,
		LessonNumber:    l.LessonNumber,
		ScheduledAt:     l.ScheduledAt,
		StartsAt:        l.StartsAt,
		EndsAt:          l.EndsAt,
		DurationMinutes: l.DurationMinutes,
		Status:          string(l.Status),
		DeliveryType:    string(l.DeliveryType),
		LocationName:    l.LocationName,
		LocationLat:     l.LocationLat,
		LocationLng:     l.LocationLng,
		GeofenceRadiusM: l.GeofenceRadiusM,
		VideoURL:        l.VideoURL,
		MaterialsURL:    l.MaterialsURL,
		Duration:        l.Duration,
		IsFree:          l.IsFree,
		CreatedAt:       l.CreatedAt,
		UpdatedAt:       l.UpdatedAt,
	}
}


// UpdateLessonMaterialsRequest represents a request to update lesson materials
type UpdateLessonMaterialsRequest struct {
	VideoURL      *string `json:"videoUrl"`      // Cloudinary video URL
	VideoPublicID *string `json:"videoPublicId"` // Cloudinary public ID
	MaterialsURL  *string `json:"materialsUrl"`  // Additional materials URL
	Duration      *int    `json:"duration"`      // Video duration in seconds
}
