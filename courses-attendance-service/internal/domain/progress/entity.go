package progress

import (
	"time"

	"github.com/google/uuid"
)

// ProgressSnapshot represents a student's progress in a course
type ProgressSnapshot struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	CourseID  uuid.UUID `gorm:"type:uuid;not null"`
	StudentID uuid.UUID `gorm:"type:uuid;not null"`

	// Progress metrics
	TotalLessons     int `gorm:"not null;default:0"`
	CompletedLessons int `gorm:"not null;default:0"`

	// Attendance breakdown
	PresentCount int `gorm:"not null;default:0"`
	LateCount    int `gorm:"not null;default:0"`
	AbsentCount  int `gorm:"not null;default:0"`
	ExcusedCount int `gorm:"not null;default:0"`

	// Calculated scores
	CompletionRatio float64 `gorm:"type:decimal(5,4);not null;default:0.0000"`
	AttendanceRatio float64 `gorm:"type:decimal(5,4);not null;default:0.0000"`
	OverallProgress float64 `gorm:"type:decimal(5,2);not null;default:0.00"`

	CalculatedAt time.Time `gorm:"type:timestamptz;not null;default:now()"`
}

func (ProgressSnapshot) TableName() string {
	return "progress_snapshots"
}

// Calculate computes progress based on attendance weight
func (p *ProgressSnapshot) Calculate(attendanceWeight float64) {
	if p.TotalLessons > 0 {
		p.CompletionRatio = float64(p.CompletedLessons) / float64(p.TotalLessons)
	}

	// Weighted attendance points
	attendedLessons := p.PresentCount + p.LateCount + p.ExcusedCount + p.AbsentCount
	if attendedLessons > 0 {
		points := float64(p.PresentCount)*1.0 +
			float64(p.LateCount)*0.7 +
			float64(p.ExcusedCount)*0.8 +
			float64(p.AbsentCount)*0.0
		p.AttendanceRatio = points / float64(p.TotalLessons)
	}

	// Overall progress formula
	p.OverallProgress = ((1-attendanceWeight)*p.CompletionRatio + attendanceWeight*p.AttendanceRatio) * 100
}
