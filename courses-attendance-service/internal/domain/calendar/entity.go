package calendar

import (
	"time"

	"github.com/OmarrGhorab/courses-attendance-service/internal/domain/lesson"
	"github.com/google/uuid"
)

// CalendarEvent represents a lesson event for calendar display
type CalendarEvent struct {
	ID          uuid.UUID           `json:"id"`
	CourseID    uuid.UUID           `json:"courseId"`
	CourseTitle string              `json:"courseTitle"`
	LessonID    uuid.UUID           `json:"lessonId"`
	Title       string              `json:"title"`
	Description string              `json:"description,omitempty"`
	Status      lesson.LessonStatus `json:"status"`

	// Timing
	ScheduledAt     time.Time  `json:"scheduledAt"`
	StartsAt        *time.Time `json:"startsAt,omitempty"`
	EndsAt          *time.Time `json:"endsAt,omitempty"`
	DurationMinutes int        `json:"durationMinutes"`

	// Location
	LocationName string   `json:"locationName,omitempty"`
	LocationLat  *float64 `json:"locationLat,omitempty"`
	LocationLng  *float64 `json:"locationLng,omitempty"`
	IsOnline     bool     `json:"isOnline"`
}

// CalendarFeed represents a user's calendar feed
type CalendarFeed struct {
	UserID    uuid.UUID       `json:"userId"`
	Events    []CalendarEvent `json:"events"`
	FetchedAt time.Time       `json:"fetchedAt"`
}
