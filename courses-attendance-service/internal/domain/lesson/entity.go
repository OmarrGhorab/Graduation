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

// DeliveryType represents how a lesson is delivered
type DeliveryType string

const (
	DeliveryTypeOnline  DeliveryType = "ONLINE"
	DeliveryTypeOffline DeliveryType = "OFFLINE"
)

// Lesson represents a single lesson within a course
type Lesson struct {
	ID       uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	CourseID uuid.UUID `gorm:"type:uuid;not null"`

	// Lesson info
	Title        string `gorm:"type:varchar(255);not null"`
	Description  string `gorm:"type:text"`
	ThumbnailURL string `gorm:"type:text"`
	LessonNumber int    `gorm:"not null"`

	// Scheduling (all UTC)
	ScheduledAt     time.Time  `gorm:"type:timestamptz;not null"`
	StartsAt        *time.Time `gorm:"type:timestamptz"`
	EndsAt          *time.Time `gorm:"type:timestamptz"`
	DurationMinutes int        `gorm:"not null;default:60"`

	// Status
	Status LessonStatus `gorm:"type:lesson_status;not null;default:'SCHEDULED'"`

	// Delivery type (can differ from course default)
	DeliveryType DeliveryType `gorm:"type:varchar(20);not null;default:'OFFLINE'"`

	// Free trial support
	IsFree bool `gorm:"not null;default:false"` // True if this lesson is free (for trial)

	// Online lesson materials (for ONLINE delivery type)
	VideoURL      string `gorm:"type:text"` // Cloudinary video URL
	VideoPublicID string `gorm:"type:varchar(255)"` // Cloudinary public ID for video
	MaterialsURL  string `gorm:"type:text"` // Additional materials (PDFs, slides, etc.)
	Duration      *int   `gorm:"type:integer"` // Video duration in seconds

	// Location override (optional, required for OFFLINE lessons)
	LocationName    string   `gorm:"type:varchar(255)"`
	LocationLat     *float64 `gorm:"type:double precision"`
	LocationLng     *float64 `gorm:"type:double precision"`
	GeofenceRadiusM *int     `gorm:"type:integer"`

	// Reminders
	RemindersSent string `gorm:"type:text;not null;default:''"` // Comma-separated minutes already notified

	// Timestamps
	CreatedAt time.Time `gorm:"not null;default:now()"`
	UpdatedAt time.Time `gorm:"not null;default:now()"`
}

func (Lesson) TableName() string {
	return "lessons"
}
