package lesson

import (
	"time"

	"github.com/google/uuid"
)

// LessonStatus represents the lifecycle status of a lesson
type LessonStatus string

const (
	LessonStatusScheduled LessonStatus = "SCHEDULED"
	LessonStatusLive      LessonStatus = "LIVE"
	LessonStatusCompleted LessonStatus = "COMPLETED"
	LessonStatusCanceled  LessonStatus = "CANCELED"
)

// Lesson represents a single lesson within a course
type Lesson struct {
	ID       uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	CourseID uuid.UUID `gorm:"type:uuid;not null"`

	// Lesson info
	Title        string `gorm:"type:varchar(255);not null"`
	Description  string `gorm:"type:text"`
	LessonNumber int    `gorm:"not null"`

	// Scheduling (all UTC)
	ScheduledAt     time.Time  `gorm:"type:timestamptz;not null"`
	StartsAt        *time.Time `gorm:"type:timestamptz"`
	EndsAt          *time.Time `gorm:"type:timestamptz"`
	DurationMinutes int        `gorm:"not null;default:60"`

	// Status
	Status LessonStatus `gorm:"type:lesson_status;not null;default:'SCHEDULED'"`

	// Location override (optional)
	LocationName    string   `gorm:"type:varchar(255)"`
	LocationLat     *float64 `gorm:"type:double precision"`
	LocationLng     *float64 `gorm:"type:double precision"`
	GeofenceRadiusM *int     `gorm:"type:integer"`

	// Timestamps
	CreatedAt time.Time `gorm:"not null;default:now()"`
	UpdatedAt time.Time `gorm:"not null;default:now()"`
}

func (Lesson) TableName() string {
	return "lessons"
}
