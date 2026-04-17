package watchtime

import (
	"math"
	"time"

	"github.com/google/uuid"
)

// DeviceType represents the device used for watching
type DeviceType string

const (
	DeviceTypeMobile  DeviceType = "MOBILE"
	DeviceTypeDesktop DeviceType = "DESKTOP"
	DeviceTypeTablet  DeviceType = "TABLET"
)

// CompletionThreshold is the percentage of video watched to count as "completed"
const CompletionThreshold = 90.0

// PreviewWatchEvent represents a single heartbeat for preview/trailer videos (before purchase)
type PreviewWatchEvent struct {
	ID             uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	CourseID       uuid.UUID  `gorm:"type:uuid;not null"` // Course being previewed
	UserID         uuid.UUID  `gorm:"type:uuid;not null"`
	WatchedSeconds int        `gorm:"not null;default:0"`
	LastPosition   int        `gorm:"not null;default:0"`
	Completed      bool       `gorm:"not null;default:false"`
	DeviceType     DeviceType `gorm:"type:varchar(20);not null;default:'DESKTOP'"`
	CreatedAt      time.Time  `gorm:"not null;default:now()"`
}

func (PreviewWatchEvent) TableName() string {
	return "preview_watch_events"
}

// UserPreviewProgress represents aggregated preview watch progress for a user on a specific course
type UserPreviewProgress struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	CourseID       uuid.UUID `gorm:"type:uuid;not null"`
	UserID         uuid.UUID `gorm:"type:uuid;not null"`
	TotalWatchTime int       `gorm:"not null;default:0"`
	MaxPosition    int       `gorm:"not null;default:0"`
	WatchCount     int       `gorm:"not null;default:0"`
	CompletionPct  float64   `gorm:"type:decimal(5,2);not null;default:0.00"`
	IsCompleted    bool      `gorm:"not null;default:false"`
	FirstWatchedAt time.Time `gorm:"type:timestamptz;not null;default:now()"`
	LastWatchedAt  time.Time `gorm:"type:timestamptz;not null;default:now()"`
	CreatedAt      time.Time `gorm:"not null;default:now()"`
	UpdatedAt      time.Time `gorm:"not null;default:now()"`
}

func (UserPreviewProgress) TableName() string {
	return "user_preview_progress"
}

// UpdateFromEvent merges a new preview watch event into this aggregated progress record.
func (p *UserPreviewProgress) UpdateFromEvent(event *PreviewWatchEvent, videoDurationSec int, now time.Time) {
	p.TotalWatchTime += event.WatchedSeconds
	p.WatchCount++
	p.LastWatchedAt = now
	p.UpdatedAt = now

	// Track the furthest point reached
	if event.LastPosition > p.MaxPosition {
		p.MaxPosition = event.LastPosition
	}

	// Calculate completion percentage
	if videoDurationSec > 0 {
		pct := (float64(p.MaxPosition) / float64(videoDurationSec)) * 100
		p.CompletionPct = math.Min(pct, 100.0)
	}

	// Mark completed if threshold reached or client signals completion
	if event.Completed || p.CompletionPct >= CompletionThreshold {
		p.IsCompleted = true
		p.CompletionPct = 100.0
	}
}

// LessonWatchEvent represents a single heartbeat/ping from the video player
type LessonWatchEvent struct {
	ID             uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	LessonID       uuid.UUID  `gorm:"type:uuid;not null"`
	UserID         uuid.UUID  `gorm:"type:uuid;not null"`
	WatchedSeconds int        `gorm:"not null;default:0"`
	LastPosition   int        `gorm:"not null;default:0"`
	Completed      bool       `gorm:"not null;default:false"`
	DeviceType     DeviceType `gorm:"type:varchar(20);not null;default:'DESKTOP'"`
	CreatedAt      time.Time  `gorm:"not null;default:now()"`
}

func (LessonWatchEvent) TableName() string {
	return "lesson_watch_events"
}

// UserLessonProgress represents aggregated watch progress for a user on a specific lesson
type UserLessonProgress struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	LessonID       uuid.UUID `gorm:"type:uuid;not null"`
	UserID         uuid.UUID `gorm:"type:uuid;not null"`
	TotalWatchTime int       `gorm:"not null;default:0"`
	MaxPosition    int       `gorm:"not null;default:0"`
	WatchCount     int       `gorm:"not null;default:0"`
	CompletionPct  float64   `gorm:"type:decimal(5,2);not null;default:0.00"`
	IsCompleted    bool      `gorm:"not null;default:false"`
	FirstWatchedAt time.Time `gorm:"type:timestamptz;not null;default:now()"`
	LastWatchedAt  time.Time `gorm:"type:timestamptz;not null;default:now()"`
	CreatedAt      time.Time `gorm:"not null;default:now()"`
	UpdatedAt      time.Time `gorm:"not null;default:now()"`
}

func (UserLessonProgress) TableName() string {
	return "user_lesson_progress"
}

// UpdateFromEvent merges a new watch event into this aggregated lesson progress record.
// videoDurationSec is the total duration of the lesson video in seconds.
func (p *UserLessonProgress) UpdateFromEvent(event *LessonWatchEvent, videoDurationSec int, now time.Time) {
	p.TotalWatchTime += event.WatchedSeconds
	p.WatchCount++
	p.LastWatchedAt = now
	p.UpdatedAt = now

	// Track the furthest point reached
	if event.LastPosition > p.MaxPosition {
		p.MaxPosition = event.LastPosition
	}

	// Calculate completion percentage
	if videoDurationSec > 0 {
		pct := (float64(p.MaxPosition) / float64(videoDurationSec)) * 100
		p.CompletionPct = math.Min(pct, 100.0)
	}

	// Mark completed if threshold reached or client signals completion
	if event.Completed || p.CompletionPct >= CompletionThreshold {
		p.IsCompleted = true
		p.CompletionPct = 100.0
	}
}

// UserCourseAnalytics represents aggregated course-level engagement for a user
type UserCourseAnalytics struct {
	ID                uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	CourseID          uuid.UUID `gorm:"type:uuid;not null"`
	UserID            uuid.UUID `gorm:"type:uuid;not null"`
	TotalWatchTime    int       `gorm:"not null;default:0"`
	LessonsStarted    int       `gorm:"not null;default:0"`
	LessonsCompleted  int       `gorm:"not null;default:0"`
	TotalLessons      int       `gorm:"not null;default:0"`
	CompletionPct     float64   `gorm:"type:decimal(5,2);not null;default:0.00"`
	AvgLessonWatchPct float64   `gorm:"type:decimal(5,2);not null;default:0.00"`
	EngagementScore   float64   `gorm:"type:decimal(5,2);not null;default:0.00"`
	LastActivityAt    time.Time `gorm:"type:timestamptz;not null;default:now()"`
	CreatedAt         time.Time `gorm:"not null;default:now()"`
	UpdatedAt         time.Time `gorm:"not null;default:now()"`
}

func (UserCourseAnalytics) TableName() string {
	return "user_course_analytics"
}

// Recompute recalculates course-level analytics from all lesson progress records.
func (a *UserCourseAnalytics) Recompute(lessonProgresses []UserLessonProgress, totalCourseLessons int, now time.Time) {
	a.TotalLessons = totalCourseLessons
	a.UpdatedAt = now

	totalWatch := 0
	started := 0
	completed := 0
	totalPct := 0.0
	var lastActivity time.Time

	for _, lp := range lessonProgresses {
		totalWatch += lp.TotalWatchTime
		if lp.WatchCount > 0 {
			started++
		}
		if lp.IsCompleted {
			completed++
		}
		totalPct += lp.CompletionPct

		if lp.LastWatchedAt.After(lastActivity) {
			lastActivity = lp.LastWatchedAt
		}
	}

	a.TotalWatchTime = totalWatch
	a.LessonsStarted = started
	a.LessonsCompleted = completed

	if totalCourseLessons > 0 {
		a.CompletionPct = (float64(completed) / float64(totalCourseLessons)) * 100
	}

	if len(lessonProgresses) > 0 {
		a.AvgLessonWatchPct = totalPct / float64(len(lessonProgresses))
	}

	if !lastActivity.IsZero() {
		a.LastActivityAt = lastActivity
	}

	// Calculate engagement score
	a.EngagementScore = a.calculateEngagementScore(now)
}

// calculateEngagementScore computes a weighted composite score.
// Formula:
//
//	0.35 * completion_pct (normalized 0-100)
//	0.25 * normalized_watch_time (capped at 100)
//	0.20 * recency_factor (decays over 30 days)
//	0.20 * rewatch_factor (rewards revisiting content)
func (a *UserCourseAnalytics) calculateEngagementScore(now time.Time) float64 {
	// Completion component
	completionComponent := a.CompletionPct

	// Watch time component: normalize against expected total
	// Assume avg lesson = 30 min; cap normalization at total_lessons * 30 * 60 seconds
	expectedWatchSec := float64(a.TotalLessons) * 30 * 60
	watchComponent := 0.0
	if expectedWatchSec > 0 {
		watchComponent = math.Min((float64(a.TotalWatchTime)/expectedWatchSec)*100, 100)
	}

	// Recency component: 100 if watched today, decays to 0 over 30 days
	daysSinceActivity := now.Sub(a.LastActivityAt).Hours() / 24
	recencyComponent := math.Max(0, 100-((daysSinceActivity/30)*100))

	// Rewatch component: rewards revisiting (capped at 100)
	rewatchRatio := 0.0
	if a.TotalLessons > 0 && a.LessonsStarted > 0 {
		// Average watches per started lesson (1 = normal, >1 = rewatching)
		avgWatches := float64(a.TotalWatchTime) / math.Max(1, float64(a.LessonsStarted)*30*60)
		rewatchRatio = math.Min(avgWatches*100, 100)
	}

	score := (0.35 * completionComponent) +
		(0.25 * watchComponent) +
		(0.20 * recencyComponent) +
		(0.20 * rewatchRatio)

	return math.Round(score*100) / 100 // Round to 2 decimals
}
