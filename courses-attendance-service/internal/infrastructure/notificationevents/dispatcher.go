package notificationevents

import (
	"context"
	"log"

	"github.com/OmarrGhorab/courses-attendance-service/internal/domain/events"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/clock"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/messaging/kafka"
	"github.com/google/uuid"
)

// EventDispatcher abstracts Kafka event submission
type EventDispatcher struct {
	producer *kafka.Producer
	clock    clock.Clock
}

func NewEventDispatcher(producer *kafka.Producer, clk clock.Clock) *EventDispatcher {
	return &EventDispatcher{
		producer: producer,
		clock:    clk,
	}
}

func (d *EventDispatcher) Dispatch(ctx context.Context, topic string, aggregateID string, actorID uuid.UUID, payload interface{}) {
	envelope := events.EventEnvelope{
		EventID:     uuid.New(),
		EventType:   topic,
		OccurredAt:  d.clock.Now(),
		AggregateID: aggregateID,
		ActorUserID: actorID,
		Payload:     payload,
	}

	err := d.producer.Publish(ctx, topic, aggregateID, envelope)
	if err != nil {
		log.Printf("Failed to dispatch event to topic %s: %v", topic, err)
	}
}

// Helper for Notification Requests
func (d *EventDispatcher) EmitNotificationRequested(ctx context.Context, actorID uuid.UUID, payload events.NotificationRequestedPayload) {
	d.Dispatch(ctx, events.TypeNotificationReq, payload.RecipientID.String(), actorID, payload)
}

// Helper for Lesson Started
func (d *EventDispatcher) EmitLessonStarted(ctx context.Context, payload events.LessonStartedPayload) {
	d.Dispatch(ctx, events.TypeLessonStarted, payload.LessonID.String(), uuid.UUID{}, payload)
}

// Helper for Lesson Ended
func (d *EventDispatcher) EmitLessonEnded(ctx context.Context, payload events.LessonEndedPayload) {
	d.Dispatch(ctx, events.TypeLessonEnded, payload.LessonID.String(), uuid.UUID{}, payload)
}

// Helper for Attendance Recorded
func (d *EventDispatcher) EmitAttendanceRecorded(ctx context.Context, actorID uuid.UUID, payload events.AttendanceRecordedPayload) {
	d.Dispatch(ctx, events.TypeAttendanceRecorded, payload.LessonID.String(), actorID, payload)
}

// Helper for Absence Requested
func (d *EventDispatcher) EmitAbsenceRequested(ctx context.Context, payload events.AbsenceRequestedPayload) {
	d.Dispatch(ctx, events.TypeAbsenceRequested, payload.RequestID.String(), payload.StudentID, payload)
}

// Helper for Absence Reviewed
func (d *EventDispatcher) EmitAbsenceReviewed(ctx context.Context, payload events.AbsenceReviewedPayload) {
	d.Dispatch(ctx, events.TypeAbsenceReviewed, payload.RequestID.String(), payload.ReviewerID, payload)
}

// Helper for Progress Updated
func (d *EventDispatcher) EmitProgressUpdated(ctx context.Context, payload events.ProgressUpdatedPayload) {
	d.Dispatch(ctx, events.TypeProgressUpdated, payload.StudentID.String(), uuid.UUID{}, payload)
}

// Helper for Attendance Fraud Detection
func (d *EventDispatcher) EmitAttendanceFraudDetected(ctx context.Context, payload events.AttendanceFraudDetectedPayload) {
	d.Dispatch(ctx, events.TypeAttendanceFraudDetected, payload.LessonID.String(), payload.StudentID, payload)
}

// Helper for Lesson Video Ready
func (d *EventDispatcher) EmitLessonVideoReady(ctx context.Context, payload events.LessonVideoReadyPayload) {
	d.Dispatch(ctx, events.TypeLessonVideoReady, payload.LessonID.String(), payload.TeacherID, payload)
}

// Helper for Lesson Video Failed
func (d *EventDispatcher) EmitLessonVideoFailed(ctx context.Context, payload events.LessonVideoFailedPayload) {
	d.Dispatch(ctx, events.TypeLessonVideoFailed, payload.LessonID.String(), payload.TeacherID, payload)
}

// Helper for Lesson Reminder
func (d *EventDispatcher) EmitLessonReminder(ctx context.Context, payload events.LessonReminderPayload) {
	d.Dispatch(ctx, events.TypeLessonReminder, payload.LessonID.String(), uuid.UUID{}, payload)
}
