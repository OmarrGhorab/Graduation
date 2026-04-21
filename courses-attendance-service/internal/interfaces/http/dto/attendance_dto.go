package dto

import (
	"time"

	"github.com/OmarrGhorab/courses-attendance-service/internal/domain/attendance"
	"github.com/google/uuid"
)

// ========== QR Token DTOs ==========

type QRTokenResponse struct {
	LessonID  uuid.UUID `json:"lessonId"`
	Payload   string    `json:"payload"`
	Signature string    `json:"signature"`
	IssuedAt  time.Time `json:"issuedAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// ========== Scan DTOs ==========

type ScanAttendanceRequest struct {
	QRPayload         string   `json:"qrPayload" validate:"required"`
	QRSignature       string   `json:"qrSignature" validate:"required"`
	DeviceID          string   `json:"deviceId" validate:"required"`
	DeviceFingerprint string   `json:"deviceFingerprint"`
	AttestationToken  string   `json:"attestationToken"`
	Latitude          *float64 `json:"latitude"`
	Longitude         *float64 `json:"longitude"`
}

type ScanAttendanceResponse struct {
	Status    string    `json:"status"`
	ScannedAt time.Time `json:"scannedAt"`
	Distance  *float64  `json:"distance,omitempty"`
	Message   string    `json:"message"`
}

// ========== Attendance Record DTOs ==========

type AttendanceRecordResponse struct {
	ID                   uuid.UUID  `json:"id"`
	LessonID             uuid.UUID  `json:"lessonId"`
	StudentID            uuid.UUID  `json:"studentId"`
	StudentName          string     `json:"studentName,omitempty"`
	StudentProfileImg    string     `json:"studentProfileImg,omitempty"`
	Status               string     `json:"status"`
	ScannedAt            *time.Time `json:"scannedAt,omitempty"`
	DistanceFromLocation *float64   `json:"distanceFromLocation,omitempty"`
	IsManualOverride     bool       `json:"isManualOverride"`
	LessonTitle          string     `json:"lessonTitle,omitempty"`
	CourseTitle          string     `json:"courseTitle,omitempty"`
	CreatedAt            time.Time  `json:"createdAt"`
	UpdatedAt            time.Time  `json:"updatedAt"`
}

func ToAttendanceRecordResponse(r *attendance.AttendanceRecord) AttendanceRecordResponse {
	return AttendanceRecordResponse{
		ID:                   r.ID,
		LessonID:             r.LessonID,
		StudentID:            r.StudentID,
		Status:               string(r.Status),
		ScannedAt:            r.ScannedAt,
		DistanceFromLocation: r.DistanceFromLocationM,
		IsManualOverride:     r.IsManualOverride,
		CreatedAt:            r.CreatedAt,
		UpdatedAt:            r.UpdatedAt,
	}
}

// ========== Attendance Session DTOs ==========

type AttendanceSessionResponse struct {
	ID         uuid.UUID  `json:"id"`
	LessonID   uuid.UUID  `json:"lessonId"`
	StartedAt  time.Time  `json:"startedAt"`
	EndedAt    *time.Time `json:"endedAt,omitempty"`
	IsActive   bool       `json:"isActive"`
	TotalScans int        `json:"totalScans"`
}

func ToAttendanceSessionResponse(s *attendance.AttendanceSession) AttendanceSessionResponse {
	return AttendanceSessionResponse{
		ID:         s.ID,
		LessonID:   s.LessonID,
		StartedAt:  s.StartedAt,
		EndedAt:    s.EndedAt,
		IsActive:   s.IsActive,
		TotalScans: s.TotalScans,
	}
}

// ========== Manual Override DTOs ==========

type ManualOverrideRequest struct {
	StudentID string `json:"studentId" validate:"required,uuid"`
	Status    string `json:"status" validate:"required,oneof=PRESENT LATE ABSENT EXCUSED"`
	Reason    string `json:"reason" validate:"required,min=5"`
}

type LessonAttendanceAnalyticsResponse struct {
	LessonID       uuid.UUID               `json:"lessonId"`
	LessonTitle    string                  `json:"lessonTitle"`
	TotalStudents  int                     `json:"totalStudents"`
	PresentCount   int                     `json:"presentCount"`
	LateCount      int                     `json:"lateCount"`
	AbsentCount    int                     `json:"absentCount"`
	ExcusedCount   int                     `json:"excusedCount"`
	AttendanceRate float64                 `json:"attendanceRate"`
	RecentActivity []RecentStudentActivity `json:"recentActivity"`
}

type RecentStudentActivity struct {
	StudentID   string     `json:"studentId"`
	StudentName string     `json:"studentName"`
	Status      string     `json:"status"`
	ScannedAt   *time.Time `json:"scannedAt"`
}
