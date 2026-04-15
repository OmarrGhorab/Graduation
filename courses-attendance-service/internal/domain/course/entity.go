package course

import (
	"time"

	"github.com/google/uuid"
)

// DeliveryType represents how a course is delivered
type DeliveryType string

const (
	DeliveryTypeOnline  DeliveryType = "ONLINE"
	DeliveryTypeOffline DeliveryType = "OFFLINE"
	DeliveryTypeHybrid  DeliveryType = "HYBRID"
)

type BillingType string

const (
	BillingTypeOneTime BillingType = "ONE_TIME"
	BillingTypeMonthly BillingType = "MONTHLY"
)

// CourseStatus represents the lifecycle status of a course
type CourseStatus string

const (
	CourseStatusActive   CourseStatus = "ACTIVE"
	CourseStatusPaused   CourseStatus = "PAUSED"
	CourseStatusArchived CourseStatus = "ARCHIVED"
)

// Course represents a course entity
type Course struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Title       string    `gorm:"type:varchar(255);not null"`
	Description string    `gorm:"type:text"`
	SubjectID   uuid.UUID `gorm:"type:uuid;not null"`
	TeacherID   uuid.UUID `gorm:"type:uuid;not null"`
	CourseImage string    `gorm:"type:text"`
	PreviewVideoURL      string    `gorm:"type:text"`
	PreviewVideoPublicID string    `gorm:"type:text"`

	// Delivery and location
	DeliveryType    DeliveryType `gorm:"type:delivery_type;not null;default:'OFFLINE'"`
	LocationName    string       `gorm:"type:varchar(255)"`
	LocationLat     *float64     `gorm:"type:double precision"`
	LocationLng     *float64     `gorm:"type:double precision"`
	GeofenceRadiusM int          `gorm:"default:100"`

	// Scheduling
	TotalLessons            int `gorm:"not null;default:0"`
	AttendanceWindowMinutes int `gorm:"not null;default:15"`

	// Pricing
	Price       float64     `gorm:"type:decimal(10,2);default:0.00"`
	Currency    string      `gorm:"type:varchar(10);not null;default:'EGP'"`
	IsPaid      bool        `gorm:"not null;default:false"`
	BillingType BillingType `gorm:"type:billing_type;not null;default:'ONE_TIME'"`
	
	// Free trial support
	FreeTrialLessons int `gorm:"not null;default:0"` // Number of free lessons (0 = all paid or all free based on IsPaid)

	// Status
	Status CourseStatus `gorm:"type:course_status;not null;default:'ACTIVE'"`

	// Progress settings
	AttendanceWeight float64 `gorm:"type:decimal(3,2);not null;default:0.30"`

	// Timestamps
	CreatedAt time.Time `gorm:"not null;default:now()"`
	UpdatedAt time.Time `gorm:"not null;default:now()"`

	// Virtual fields (calculated)
	EnrollmentCount int `gorm:"-"`

	// Relations
	Subject    *Subject          `gorm:"foreignKey:SubjectID"`
	Assistants []CourseAssistant `gorm:"foreignKey:CourseID"`
}

func (Course) TableName() string {
	return "courses"
}

// Subject represents a course category
type Subject struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name        string    `gorm:"type:varchar(100);not null;unique"`
	Description string    `gorm:"type:text"`
	Icon        string    `gorm:"type:varchar(100)"`
	CreatedAt   time.Time `gorm:"not null;default:now()"`
	UpdatedAt   time.Time `gorm:"not null;default:now()"`
}

func (Subject) TableName() string {
	return "subjects"
}

// CourseAssistant represents a teacher assistant assigned to a course
type CourseAssistant struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	CourseID    uuid.UUID `gorm:"type:uuid;not null"`
	AssistantID uuid.UUID `gorm:"type:uuid;not null"`

	// Permissions
	CanStartLesson    bool `gorm:"not null;default:true"`
	CanEndLesson      bool `gorm:"not null;default:true"`
	CanViewAttendance bool `gorm:"not null;default:true"`
	CanEditAttendance bool `gorm:"not null;default:false"`

	CreatedAt time.Time `gorm:"not null;default:now()"`
}

func (CourseAssistant) TableName() string {
	return "course_assistants"
}

// Enrollment represents a student enrolled in a course
type Enrollment struct {
	ID       uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	CourseID uuid.UUID `gorm:"type:uuid;not null"`
	UserID   uuid.UUID `gorm:"type:uuid;not null"`

	IsActive bool       `gorm:"not null;default:true"`
	IsPaid   bool       `gorm:"not null;default:false"`
	PaidAt   *time.Time `gorm:"type:timestamptz"`

	EnrolledAt time.Time `gorm:"not null;default:now()"`
	UpdatedAt  time.Time `gorm:"not null;default:now()"`
}

func (Enrollment) TableName() string {
	return "enrollments"
}

// EnrollmentPeriod represents a specifically paid month for an enrollment
type EnrollmentPeriod struct {
	ID           uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	EnrollmentID uuid.UUID  `gorm:"type:uuid;not null"`
	PeriodKey    string     `gorm:"type:varchar(10);not null"` // Format: YYYY-MM
	IsPaid       bool       `gorm:"not null;default:false"`
	PaidAt       *time.Time `gorm:"type:timestamptz"`
	CreatedAt    time.Time  `gorm:"not null;default:now()"`
	UpdatedAt    time.Time  `gorm:"not null;default:now()"`
}

func (EnrollmentPeriod) TableName() string {
	return "enrollment_periods"
}

