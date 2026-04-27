package absence

import (
	"time"

	"github.com/google/uuid"
)

// AbsenceReasonType represents the category of absence
type AbsenceReasonType string

const (
	AbsenceReasonParentExcuse AbsenceReasonType = "PARENT_EXCUSE"
	AbsenceReasonMedical      AbsenceReasonType = "MEDICAL"
	AbsenceReasonEmergency    AbsenceReasonType = "EMERGENCY"
	AbsenceReasonTechnical    AbsenceReasonType = "TECHNICAL"
	AbsenceReasonPersonal     AbsenceReasonType = "PERSONAL"
)

// AbsenceStatus represents the approval state of an absence request
type AbsenceStatus string

const (
	AbsenceStatusPending  AbsenceStatus = "PENDING"
	AbsenceStatusApproved AbsenceStatus = "APPROVED"
	AbsenceStatusRejected AbsenceStatus = "REJECTED"
)

// AbsenceRequest represents a request to excuse an absence
type AbsenceRequest struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	LessonID  uuid.UUID `gorm:"type:uuid;not null"`
	StudentID uuid.UUID `gorm:"type:uuid;not null"`

	// Request details
	ReasonType AbsenceReasonType `gorm:"type:absence_reason_type;not null"`
	ReasonText string            `gorm:"type:text"`

	// Attachments
	AttachmentURL string `gorm:"type:text"`

	// Requester info
	RequestedBy uuid.UUID `gorm:"type:uuid;not null"`
	RequestedAt time.Time `gorm:"type:timestamptz;not null;default:now()"`

	// Response
	Status       AbsenceStatus `gorm:"type:absence_status;not null;default:'PENDING'"`
	RespondedBy  *uuid.UUID    `gorm:"type:uuid"`
	RespondedAt  *time.Time    `gorm:"type:timestamptz"`
	ResponseNote string        `gorm:"type:text"`

	// Link to attendance record
	AttendanceRecordID *uuid.UUID `gorm:"type:uuid"`

	CreatedAt time.Time `gorm:"not null;default:now()"`
	UpdatedAt time.Time `gorm:"not null;default:now()"`
}

func (AbsenceRequest) TableName() string {
	return "absence_requests"
}
