package dto

import (
	"time"

	"github.com/OmarrGhorab/courses-attendance-service/internal/domain/watchtime"
	"github.com/google/uuid"
)

// --- Requests ---

// WatchHeartbeatRequest is sent by the client every ~60 seconds (or on events like pause/seek)
type WatchHeartbeatRequest struct {
	LessonID       string `json:"lessonId" validate:"required,uuid"`
	WatchedSeconds int    `json:"watchedSeconds" validate:"required,min=1,max=300"`
	LastPosition   int    `json:"lastPosition" validate:"required,min=0"`
	Completed      bool   `json:"completed"`
	DeviceType     string `json:"deviceType" validate:"omitempty,oneof=MOBILE DESKTOP TABLET"`
}

// PreviewHeartbeatRequest is sent by the client for preview/trailer videos (before purchase)
type PreviewHeartbeatRequest struct {
	CourseID       string `json:"courseId" validate:"required,uuid"`
	WatchedSeconds int    `json:"watchedSeconds" validate:"required,min=1,max=300"`
	LastPosition   int    `json:"lastPosition" validate:"required,min=0"`
	Completed      bool   `json:"completed"`
	DeviceType     string `json:"deviceType" validate:"omitempty,oneof=MOBILE DESKTOP TABLET"`
}

// --- Responses ---

// LessonProgressResponse represents a student's watch progress on a single lesson
type LessonProgressResponse struct {
	LessonID       uuid.UUID `json:"lessonId"`
	TotalWatchTime int       `json:"totalWatchTime"`   // seconds
	MaxPosition    int       `json:"maxPosition"`      // seconds
	WatchCount     int       `json:"watchCount"`
	CompletionPct  float64   `json:"completionPct"`    // 0-100
	IsCompleted    bool      `json:"isCompleted"`
	FirstWatchedAt time.Time `json:"firstWatchedAt"`
	LastWatchedAt  time.Time `json:"lastWatchedAt"`
}

// ToLessonProgressResponse converts domain entity to response DTO
func ToLessonProgressResponse(p *watchtime.UserLessonProgress) LessonProgressResponse {
	return LessonProgressResponse{
		LessonID:       p.LessonID,
		TotalWatchTime: p.TotalWatchTime,
		MaxPosition:    p.MaxPosition,
		WatchCount:     p.WatchCount,
		CompletionPct:  p.CompletionPct,
		IsCompleted:    p.IsCompleted,
		FirstWatchedAt: p.FirstWatchedAt,
		LastWatchedAt:  p.LastWatchedAt,
	}
}

// CourseAnalyticsResponse represents course-level engagement analytics for a student
type CourseAnalyticsResponse struct {
	CourseID          uuid.UUID `json:"courseId"`
	CourseTitle       string    `json:"courseTitle,omitempty"`
	TotalWatchTime    int       `json:"totalWatchTime"`    // seconds
	LessonsStarted    int       `json:"lessonsStarted"`
	LessonsCompleted  int       `json:"lessonsCompleted"`
	TotalLessons      int       `json:"totalLessons"`
	CompletionPct     float64   `json:"completionPct"`     // 0-100
	AvgLessonWatchPct float64   `json:"avgLessonWatchPct"` // 0-100
	EngagementScore   float64   `json:"engagementScore"`   // 0-100
	LastActivityAt    time.Time `json:"lastActivityAt"`
}

// ToCourseAnalyticsResponse converts domain entity to response DTO
func ToCourseAnalyticsResponse(a *watchtime.UserCourseAnalytics) CourseAnalyticsResponse {
	return CourseAnalyticsResponse{
		CourseID:          a.CourseID,
		TotalWatchTime:    a.TotalWatchTime,
		LessonsStarted:    a.LessonsStarted,
		LessonsCompleted:  a.LessonsCompleted,
		TotalLessons:      a.TotalLessons,
		CompletionPct:     a.CompletionPct,
		AvgLessonWatchPct: a.AvgLessonWatchPct,
		EngagementScore:   a.EngagementScore,
		LastActivityAt:    a.LastActivityAt,
	}
}

// StudentDashboardResponse contains a student's analytics across all enrolled courses
type StudentDashboardResponse struct {
	TotalWatchTime     int                       `json:"totalWatchTime"`     // total seconds across all courses
	TotalCoursesActive int                       `json:"totalCoursesActive"`
	TotalCompleted     int                       `json:"totalCompleted"`
	OverallEngagement  float64                   `json:"overallEngagement"`  // average engagement
	Courses            []CourseAnalyticsResponse `json:"courses"`
}

// LeaderboardEntryResponse represents a single student on the engagement leaderboard
type LeaderboardEntryResponse struct {
	Rank              int       `json:"rank"`
	UserID            uuid.UUID `json:"userId"`
	TotalWatchTime    int       `json:"totalWatchTime"`
	LessonsCompleted  int       `json:"lessonsCompleted"`
	CompletionPct     float64   `json:"completionPct"`
	EngagementScore   float64   `json:"engagementScore"`
	LastActivityAt    time.Time `json:"lastActivityAt"`
}

// SubjectPreferenceResponse represents a student's engagement with a subject (for AI)
type SubjectPreferenceResponse struct {
	SubjectID        uuid.UUID `json:"subjectId"`
	SubjectName      string    `json:"subjectName"`
	TotalWatchTime   int       `json:"totalWatchTime"`
	CoursesWatched   int       `json:"coursesWatched"`
	AvgEngagement    float64   `json:"avgEngagement"`
	AvgCompletionPct float64   `json:"avgCompletionPct"`
}

// RecommendationProfileResponse is the structured data export for the AI recommendation engine
type RecommendationProfileResponse struct {
	UserID             uuid.UUID                   `json:"userId"`
	TotalWatchTime     int                         `json:"totalWatchTime"`
	TotalCoursesActive int                         `json:"totalCoursesActive"`
	TotalCompleted     int                         `json:"totalCompleted"`
	OverallEngagement  float64                     `json:"overallEngagement"`
	SubjectPreferences []SubjectPreferenceResponse `json:"subjectPreferences"`
	PreviewInterests   []PreviewInterestResponse   `json:"previewInterests"`   // NEW: Preview video interests
	CourseAnalytics    []CourseAnalyticsResponse   `json:"courseAnalytics"`
	PreviewProgress    []PreviewProgressResponse   `json:"previewProgress"`    // NEW: All preview progress
	WatchPatterns      WatchPatternsResponse       `json:"watchPatterns"`
}

// WatchPatternsResponse contains behavioral patterns extracted from watch data
type WatchPatternsResponse struct {
	AvgSessionDuration  int     `json:"avgSessionDuration"`    // average seconds per watch session
	PreferredDeviceType string  `json:"preferredDeviceType"`   // most used device
	CompletionTendency  string  `json:"completionTendency"`    // HIGH (>70%), MEDIUM (40-70%), LOW (<40%)
	AvgCompletionPct    float64 `json:"avgCompletionPct"`
}

// PreviewProgressResponse represents a user's watch progress on a course preview
type PreviewProgressResponse struct {
	CourseID       uuid.UUID `json:"courseId"`
	TotalWatchTime int       `json:"totalWatchTime"`   // seconds
	MaxPosition    int       `json:"maxPosition"`      // seconds
	WatchCount     int       `json:"watchCount"`
	CompletionPct  float64   `json:"completionPct"`    // 0-100
	IsCompleted    bool      `json:"isCompleted"`
	FirstWatchedAt time.Time `json:"firstWatchedAt"`
	LastWatchedAt  time.Time `json:"lastWatchedAt"`
}

// ToPreviewProgressResponse converts domain entity to response DTO
func ToPreviewProgressResponse(p *watchtime.UserPreviewProgress) PreviewProgressResponse {
	return PreviewProgressResponse{
		CourseID:       p.CourseID,
		TotalWatchTime: p.TotalWatchTime,
		MaxPosition:    p.MaxPosition,
		WatchCount:     p.WatchCount,
		CompletionPct:  p.CompletionPct,
		IsCompleted:    p.IsCompleted,
		FirstWatchedAt: p.FirstWatchedAt,
		LastWatchedAt:  p.LastWatchedAt,
	}
}

// PreviewInterestResponse represents a user's interest in a subject based on preview views
type PreviewInterestResponse struct {
	SubjectID        uuid.UUID `json:"subjectId"`
	SubjectName      string    `json:"subjectName"`
	TotalWatchTime   int       `json:"totalWatchTime"`
	CoursesViewed    int       `json:"coursesViewed"`
	AvgCompletionPct float64   `json:"avgCompletionPct"`
}
