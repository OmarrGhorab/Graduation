package events

import (
	"time"

	"github.com/google/uuid"
)

// Event types
const (
	TypeLessonStarted       = "courses.lesson.started.v1"
	TypeLessonEnded         = "courses.lesson.ended.v1"
	TypeLessonCanceled      = "courses.lesson.canceled.v1"
	TypeLessonRescheduled   = "courses.lesson.rescheduled.v1"
	TypeAttendanceRecorded  = "courses.attendance.recorded.v1"
	TypeAttendanceFinalized = "courses.attendance.finalized.v1"
	TypeAbsenceRequested    = "courses.absence.requested.v1"
	TypeAbsenceReviewed     = "courses.absence.reviewed.v1"
	TypeProgressUpdated     = "courses.progress.updated.v1"
	TypeNotificationReq     = "courses.notification.requested.v1"
	TypeAttendanceFraudDetected = "courses.attendance.fraud_detected.v1"
	TypeLessonVideoReady    = "courses.lesson.video_ready.v1"
	TypeLessonVideoFailed   = "courses.lesson.video_failed.v1"
)

// EventEnvelope represents the standard structure for all Kafka events
type EventEnvelope struct {
	EventID     uuid.UUID   `json:"event_id"`
	EventType   string      `json:"event_type"`
	OccurredAt  time.Time   `json:"occurred_at"`
	AggregateID string      `json:"aggregate_id"`
	ActorUserID uuid.UUID   `json:"actor_user_id,omitempty"`
	Payload     interface{} `json:"payload"`
	TraceID     string      `json:"trace_id,omitempty"`
}

// Payload structures

type LessonStartedPayload struct {
	LessonID uuid.UUID `json:"lesson_id"`
	CourseID uuid.UUID `json:"course_id"`
	StartsAt time.Time `json:"starts_at"`
}

type LessonEndedPayload struct {
	LessonID uuid.UUID `json:"lesson_id"`
	CourseID uuid.UUID `json:"course_id"`
	EndsAt   time.Time `json:"ends_at"`
}

type LessonCanceledPayload struct {
	LessonID uuid.UUID `json:"lesson_id"`
	CourseID uuid.UUID `json:"course_id"`
	Reason   string    `json:"reason"`
}

type LessonRescheduledPayload struct {
	LessonID       uuid.UUID `json:"lesson_id"`
	OldScheduledAt time.Time `json:"old_scheduled_at"`
	NewScheduledAt time.Time `json:"new_scheduled_at"`
}

type AttendanceRecordedPayload struct {
	LessonID    uuid.UUID `json:"lesson_id"`
	LessonTitle string    `json:"lesson_title"`
	CourseID    uuid.UUID `json:"course_id"`
	CourseTitle string    `json:"course_title"`
	StudentID   uuid.UUID `json:"student_id"`
	TeacherID   uuid.UUID `json:"teacher_id"`
	Status      string    `json:"status"`
	ScannedAt   time.Time `json:"scanned_at"`
}

type AttendanceFinalizedPayload struct {
	LessonID uuid.UUID `json:"lesson_id"`
}

type AbsenceRequestedPayload struct {
	RequestID   uuid.UUID `json:"request_id"`
	LessonID    uuid.UUID `json:"lesson_id"`
	LessonTitle string    `json:"lesson_title"`
	CourseID    uuid.UUID `json:"course_id"`
	CourseTitle string    `json:"course_title"`
	StudentID   uuid.UUID `json:"student_id"`
	TeacherID   uuid.UUID `json:"teacher_id"`
	Reason      string    `json:"reason"`
}

type AbsenceReviewedPayload struct {
	RequestID  uuid.UUID `json:"request_id"`
	Status     string    `json:"status"`
	ReviewerID uuid.UUID `json:"reviewer_id"`
}

type ProgressUpdatedPayload struct {
	CourseID        uuid.UUID `json:"course_id"`
	StudentID       uuid.UUID `json:"student_id"`
	OverallProgress float64   `json:"overall_progress"`
}

type NotificationRequestedPayload struct {
	RecipientID uuid.UUID              `json:"recipient_id"`
	Type        string                 `json:"type"`
	Data        map[string]interface{} `json:"data"`
}

type AttendanceFraudDetectedPayload struct {
	LessonID          uuid.UUID `json:"lesson_id"`
	LessonTitle       string    `json:"lesson_title"`
	CourseID          uuid.UUID `json:"course_id"`
	CourseTitle       string    `json:"course_title"`
	StudentID         uuid.UUID `json:"student_id"`
	ExistingStudentID uuid.UUID `json:"existing_student_id"`
	DeviceID          string    `json:"device_id"`
	TeacherID         uuid.UUID `json:"teacher_id"`
	DetectedAt        time.Time `json:"detected_at"`
}

type LessonVideoReadyPayload struct {
	LessonID      uuid.UUID `json:"lesson_id"`
	LessonTitle   string    `json:"lesson_title"`
	TeacherID     uuid.UUID `json:"teacher_id"`
	VideoURL      string    `json:"video_url"`
	StreamingURL  string    `json:"streaming_url"`
	Duration      int       `json:"duration"`
}

type LessonVideoFailedPayload struct {
	LessonID    uuid.UUID `json:"lesson_id"`
	LessonTitle string    `json:"lesson_title"`
	TeacherID   uuid.UUID `json:"teacher_id"`
	Error       string    `json:"error"`
}
