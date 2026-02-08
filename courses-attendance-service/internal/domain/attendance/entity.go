package attendance

import (
	"time"

	"github.com/google/uuid"
)

// AttendanceStatus represents student attendance state
type AttendanceStatus string

const (
	AttendanceStatusPresent AttendanceStatus = "PRESENT"
	AttendanceStatusLate    AttendanceStatus = "LATE"
	AttendanceStatusAbsent  AttendanceStatus = "ABSENT"
	AttendanceStatusExcused AttendanceStatus = "EXCUSED"
)

// AttendancePoints returns the weighted points for progress calculation
func (s AttendanceStatus) Points() float64 {
	switch s {
	case AttendanceStatusPresent:
		return 1.0
	case AttendanceStatusLate:
		return 0.7
	case AttendanceStatusExcused:
		return 0.8
	case AttendanceStatusAbsent:
		return 0.0
	default:
		return 0.0
	}
}

// AttendanceSession represents an active attendance session for a lesson
type AttendanceSession struct {
	ID       uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	LessonID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`

	StartedAt time.Time  `gorm:"type:timestamptz;not null;default:now()"`
	EndedAt   *time.Time `gorm:"type:timestamptz"`
	IsActive  bool       `gorm:"not null;default:true"`

	TotalScans int `gorm:"not null;default:0"`

	CreatedAt time.Time `gorm:"not null;default:now()"`
}

func (AttendanceSession) TableName() string {
	return "attendance_sessions"
}

// AttendanceQRToken represents a rotating QR code token
type AttendanceQRToken struct {
	ID       uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	LessonID uuid.UUID `gorm:"type:uuid;not null"`

	Nonce     string `gorm:"type:varchar(64);not null"`
	Payload   string `gorm:"type:text;not null"`
	Signature string `gorm:"type:varchar(128);not null"`

	IssuedAt  time.Time `gorm:"type:timestamptz;not null"`
	ExpiresAt time.Time `gorm:"type:timestamptz;not null"`

	IsConsumed bool       `gorm:"not null;default:false"`
	ConsumedBy *uuid.UUID `gorm:"type:uuid"`
	ConsumedAt *time.Time `gorm:"type:timestamptz"`
}

func (AttendanceQRToken) TableName() string {
	return "attendance_qr_tokens"
}

// AttendanceRecord represents a student's attendance for a lesson
type AttendanceRecord struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	LessonID  uuid.UUID `gorm:"type:uuid;not null"`
	StudentID uuid.UUID `gorm:"type:uuid;not null"`

	Status    AttendanceStatus `gorm:"type:attendance_status;not null;default:'ABSENT'"`
	ScannedAt *time.Time       `gorm:"type:timestamptz"`

	// Device/security info
	DeviceID          string `gorm:"type:varchar(255)"`
	DeviceFingerprint string `gorm:"type:varchar(255)"`
	IPAddress         string `gorm:"type:varchar(45)"`
	UserAgent         string `gorm:"type:text"`

	// Location data
	ScanLat               *float64 `gorm:"type:double precision"`
	ScanLng               *float64 `gorm:"type:double precision"`
	DistanceFromLocationM *float64 `gorm:"type:double precision"`

	// QR token reference
	QRTokenID *uuid.UUID `gorm:"type:uuid"`

	// Manual override
	IsManualOverride bool       `gorm:"not null;default:false"`
	OverrideBy       *uuid.UUID `gorm:"type:uuid"`
	OverrideReason   string     `gorm:"type:text"`

	CreatedAt time.Time `gorm:"not null;default:now()"`
	UpdatedAt time.Time `gorm:"not null;default:now()"`
}

func (AttendanceRecord) TableName() string {
	return "attendance_records"
}
