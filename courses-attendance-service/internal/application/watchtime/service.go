package watchtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/OmarrGhorab/courses-attendance-service/internal/domain/watchtime"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/aiclient"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/authclient"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/clock"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/persistence/postgres"
	"github.com/google/uuid"
)

var (
	ErrLessonNotFound  = errors.New("lesson not found")
	ErrCourseNotFound  = errors.New("course not found")
	ErrNotEnrolled     = errors.New("user is not enrolled in this course")
	ErrNotOnlineLesson = errors.New("watch tracking is only available for online lessons")
	ErrInvalidInput    = errors.New("invalid input data")
)


// RecordWatchInput represents the input for recording a watch event
type RecordWatchInput struct {
	LessonID       uuid.UUID
	WatchedSeconds int
	LastPosition   int
	Completed      bool
	DeviceType     watchtime.DeviceType
}

// RecordPreviewInput represents the input for recording a preview watch event
type RecordPreviewInput struct {
	CourseID       uuid.UUID
	WatchedSeconds int
	LastPosition   int
	Completed      bool
	DeviceType     watchtime.DeviceType
}

// Service handles watch time and analytics business logic
type Service struct {
	watchRepo      *postgres.WatchTimeRepository
	lessonRepo     *postgres.LessonRepository
	courseRepo     *postgres.CourseRepository
	enrollmentRepo *postgres.EnrollmentRepository
	recordRepo     *postgres.AttendanceRecordRepository
	clock          clock.Clock
	aiClient       *aiclient.Client
	authClient     *authclient.Client
	paymentURL     string
	internalSecret string
}

func NewService(
	watchRepo *postgres.WatchTimeRepository,
	lessonRepo *postgres.LessonRepository,
	courseRepo *postgres.CourseRepository,
	enrollmentRepo *postgres.EnrollmentRepository,
	recordRepo *postgres.AttendanceRecordRepository,
	clk clock.Clock,
	aiClient *aiclient.Client,
	authClient *authclient.Client,
	paymentURL string,
	internalSecret string,
) *Service {
	return &Service{
		watchRepo:      watchRepo,
		lessonRepo:     lessonRepo,
		courseRepo:     courseRepo,
		enrollmentRepo: enrollmentRepo,
		recordRepo:     recordRepo,
		clock:          clk,
		aiClient:       aiClient,
		authClient:     authClient,
		paymentURL:     paymentURL,
		internalSecret: internalSecret,
	}
}

// RecordWatchEvent validates and records a watch heartbeat, then updates aggregated progress
func (s *Service) RecordWatchEvent(ctx context.Context, userID uuid.UUID, input RecordWatchInput) (*watchtime.UserLessonProgress, error) {
	// 1. Validate lesson exists and is ONLINE
	lesson, err := s.lessonRepo.GetByID(ctx, input.LessonID)
	if err != nil {
		return nil, fmt.Errorf("failed to get lesson: %w", err)
	}
	if lesson == nil {
		return nil, ErrLessonNotFound
	}
	if lesson.DeliveryType != "ONLINE" {
		return nil, ErrNotOnlineLesson
	}

	// 2. Validate user is enrolled in the course
	enrollment, err := s.enrollmentRepo.GetByCourseAndUser(ctx, lesson.CourseID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check enrollment: %w", err)
	}
	if enrollment == nil {
		return nil, ErrNotEnrolled
	}

	now := s.clock.Now()

	// 3. Insert raw watch event
	event := &watchtime.LessonWatchEvent{
		ID:             uuid.New(),
		LessonID:       input.LessonID,
		UserID:         userID,
		WatchedSeconds: input.WatchedSeconds,
		LastPosition:   input.LastPosition,
		Completed:      input.Completed,
		DeviceType:     input.DeviceType,
		CreatedAt:      now,
	}

	if err := s.watchRepo.CreateWatchEvent(ctx, event); err != nil {
		return nil, fmt.Errorf("failed to create watch event: %w", err)
	}

	// 4. Update aggregated lesson progress
	progress, err := s.watchRepo.GetUserLessonProgress(ctx, userID, input.LessonID)
	if err != nil {
		return nil, fmt.Errorf("failed to get lesson progress: %w", err)
	}

	if progress == nil {
		// First time watching this lesson
		progress = &watchtime.UserLessonProgress{
			ID:             uuid.New(),
			LessonID:       input.LessonID,
			UserID:         userID,
			FirstWatchedAt: now,
			CreatedAt:      now,
		}
	}

	// Get video duration from lesson (in seconds)
	videoDuration := 0
	if lesson.Duration != nil {
		videoDuration = *lesson.Duration
	}

	progress.UpdateFromEvent(event, videoDuration, now)

	if err := s.watchRepo.UpsertUserLessonProgress(ctx, progress); err != nil {
		return nil, fmt.Errorf("failed to upsert lesson progress: %w", err)
	}

	// 5. Async recompute course analytics (Only on important events to save resources)
	// Trigger recompute if: 
	// 1. Lesson completed
	// 2. Large watch block (60s+) 
	// 3. Every 2 heartbeats (for short videos)
	// 4. First watch
	// 5. Short video (duration < 30s) - recompute every heartbeat
	isShortVideo := videoDuration > 0 && videoDuration < 30
	shouldRecompute := input.Completed || 
		input.WatchedSeconds >= 60 || 
		(progress.WatchCount%2 == 0) || 
		progress.WatchCount == 1 ||
		isShortVideo
	if shouldRecompute {
		go s.recomputeCourseAnalytics(lesson.CourseID, userID)
	}

	// 6. If lesson completed, invalidate AI recommendation cache so recommendations refresh in real-time
	if input.Completed {
		go s.invalidateAICache(userID.String())
	}

	return progress, nil
}

// invalidateAICache notifies the AI service to refresh recommendations for a user
func (s *Service) invalidateAICache(userID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	_ = s.aiClient.InvalidateRecommendationCache(ctx, userID)
}

// recomputeCourseAnalytics recalculates course-level analytics for a user
func (s *Service) recomputeCourseAnalytics(courseID, userID uuid.UUID) {
	ctx := context.Background()

	// Get all lesson progress for this user in this course
	progresses, err := s.watchRepo.GetUserLessonProgressByCourse(ctx, userID, courseID)
	if err != nil {
		return
	}

	// Get total lessons in course
	course, err := s.courseRepo.GetByID(ctx, courseID)
	if err != nil || course == nil {
		return
	}

	now := s.clock.Now()

	// Get or create analytics
	analytics, err := s.watchRepo.GetUserCourseAnalytics(ctx, userID, courseID)
	if err != nil {
		return
	}

	if analytics == nil {
		analytics = &watchtime.UserCourseAnalytics{
			ID:        uuid.New(),
			CourseID:  courseID,
			UserID:    userID,
			CreatedAt: now,
		}
	}

	analytics.Recompute(progresses, course.TotalLessons, now)

	_ = s.watchRepo.UpsertUserCourseAnalytics(ctx, analytics)

	// Invalidate recommendation cache so AI re-calculates on next visit
	go s.invalidateRecommendationCache(userID)
}

// invalidateRecommendationCache notifies the recommendation service to clear its cache for this user
func (s *Service) invalidateRecommendationCache(userID uuid.UUID) {
	// Simple HTTP DELETE call to the Python service
	// In production, this would use a discovery service or env var
	recUrl := os.Getenv("RECOMMENDATION_SERVICE_URL")
	if recUrl == "" {
		recUrl = "http://recommendation-service:8095" // Docker internal name
	}
	secret := os.Getenv("INTERNAL_SERVICE_SECRET")

	url := fmt.Sprintf("%s/api/v1/recommendations/cache/%s", recUrl, userID.String())

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return
	}

	req.Header.Set("x-internal-service-secret", secret)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
}

// GetLessonProgress returns a student's watch progress on a specific lesson
func (s *Service) GetLessonProgress(ctx context.Context, userID, lessonID uuid.UUID) (*watchtime.UserLessonProgress, error) {
	progress, err := s.watchRepo.GetUserLessonProgress(ctx, userID, lessonID)
	if err != nil {
		return nil, fmt.Errorf("failed to get lesson progress: %w", err)
	}
	return progress, nil
}

// GetCourseAnalytics returns a student's course-level engagement analytics
func (s *Service) GetCourseAnalytics(ctx context.Context, userID, courseID uuid.UUID) (*watchtime.UserCourseAnalytics, error) {
	analytics, err := s.watchRepo.GetUserCourseAnalytics(ctx, userID, courseID)
	if err != nil {
		return nil, fmt.Errorf("failed to get course analytics: %w", err)
	}
	return analytics, nil
}

// DashboardData holds all data for a student's dashboard
type DashboardData struct {
	AllAnalytics []watchtime.UserCourseAnalytics
}

// GetStudentDashboard returns analytics for all enrolled courses
func (s *Service) GetStudentDashboard(ctx context.Context, userID uuid.UUID) (*DashboardData, error) {
	analytics, err := s.watchRepo.GetUserAllCourseAnalytics(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get dashboard data: %w", err)
	}
	return &DashboardData{AllAnalytics: analytics}, nil
}

// GetCourseLeaderboard returns the engagement leaderboard for a course
func (s *Service) GetCourseLeaderboard(ctx context.Context, courseID uuid.UUID, limit int) ([]watchtime.UserCourseAnalytics, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.watchRepo.GetCourseEngagementLeaderboard(ctx, courseID, limit)
}

// RecommendationProfile holds data for the AI recommendation engine
type RecommendationProfile struct {
	AllAnalytics       []watchtime.UserCourseAnalytics
	SubjectPreferences []postgres.SubjectWatchStat
	PreviewInterests   []postgres.PreviewInterestStat // NEW: Preview video interests
	AllPreviewProgress []watchtime.UserPreviewProgress // NEW: All preview progress
	AvgSessionDuration int
	PreferredDevice    string
	CompletionTendency string
	AvgCompletionPct   float64
	UserInterests      []string // From Auth (onboarding)
	CartSubjects       []string // From Payment/Cart
}

// GetRecommendationProfile returns structured data for AI course recommendations
func (s *Service) GetRecommendationProfile(ctx context.Context, userID uuid.UUID) (*RecommendationProfile, error) {
	// 1. Fetch user interests from Auth Service
	var interests []string
	userInfo, err := s.authClient.GetUserInfo(ctx, userID.String())
	if err == nil && userInfo != nil {
		interests = userInfo.Interests
	}

	// 2. Fetch cart subjects from Payment Service
	var cartSubjects []string
	cartURL := fmt.Sprintf("%s/api/v1/internal/cart/%s", s.paymentURL, userID)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, cartURL, nil)
	req.Header.Set("x-internal-service-secret", s.internalSecret)
	
	pClient := &http.Client{Timeout: 5 * time.Second}
	resp, err := pClient.Do(req)
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			var result struct {
				Success bool     `json:"success"`
				Data    []string `json:"data"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
				cartSubjects = result.Data
			}
		}
	}

	// 3. Get all course analytics (Online/Video based)
	analytics, err := s.watchRepo.GetUserAllCourseAnalytics(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get course analytics: %w", err)
	}

	// 4. Merge Offline/Live Attendance Completions
	attendanceRecords, err := s.recordRepo.GetByStudentID(ctx, userID)
	if err == nil {
		// Group attendance completions by CourseID
		courseOfflineCompletions := make(map[uuid.UUID]int)
		for _, record := range attendanceRecords {
			// Status PRESENT or LATE counts as completed for an offline/live lesson
			if record.Status == "PRESENT" || record.Status == "LATE" {
				lesson, _ := s.lessonRepo.GetByID(ctx, record.LessonID)
				if lesson != nil {
					courseOfflineCompletions[lesson.CourseID]++
				}
			}
		}

		// Adjust analytics to include offline completions
		for i, a := range analytics {
			if count, exists := courseOfflineCompletions[a.CourseID]; exists {
				// We don't want to double count if the system already marked it via progress sync
				// But to be safe for Recommendations, we ensure it's at least this high
				if count > a.LessonsCompleted {
					analytics[i].LessonsCompleted = count
					// Boost completion percentage estimate for offline lessons
					// (Rough estimate: if offline is completed, we treat as 100% watched)
					totalLessons, _ := s.courseRepo.GetByID(ctx, a.CourseID)
					if totalLessons != nil && totalLessons.TotalLessons > 0 {
						analytics[i].CompletionPct = float64(count) / float64(totalLessons.TotalLessons) * 100
					}
				}
			}
		}
	}

	// Get subject preferences
	subjects, err := s.watchRepo.GetMostWatchedSubjects(ctx, userID, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to get subject preferences: %w", err)
	}

	// Get preview interests (courses user watched but didn't purchase)
	previewInterests, err := s.watchRepo.GetPreviewInterestsBySubject(ctx, userID, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to get preview interests: %w", err)
	}

	// Get all preview progress
	allPreviewProgress, err := s.watchRepo.GetAllPreviewProgressForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get preview progress: %w", err)
	}

	// Calculate watch patterns
	totalWatchTime := 0
	totalCompleted := 0
	totalCompletionPct := 0.0
	for _, a := range analytics {
		totalWatchTime += a.TotalWatchTime
		totalCompleted += a.LessonsCompleted
		totalCompletionPct += a.AvgLessonWatchPct
	}

	avgCompletionPct := 0.0
	if len(analytics) > 0 {
		avgCompletionPct = totalCompletionPct / float64(len(analytics))
	}

	// Determine completion tendency
	tendency := "LOW"
	if avgCompletionPct >= 70 {
		tendency = "HIGH"
	} else if avgCompletionPct >= 40 {
		tendency = "MEDIUM"
	}

	// Get all lesson progress for device preference
	allProgress, err := s.watchRepo.GetAllLessonProgressForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get lesson progress: %w", err)
	}

	avgSession := 0
	if len(allProgress) > 0 {
		totalSessions := 0
		for _, p := range allProgress {
			totalSessions += p.WatchCount
		}
		if totalSessions > 0 {
			avgSession = totalWatchTime / totalSessions
		}
	}

	return &RecommendationProfile{
		AllAnalytics:       analytics,
		SubjectPreferences: subjects,
		PreviewInterests:   previewInterests,
		AllPreviewProgress: allPreviewProgress,
		AvgSessionDuration: avgSession,
		PreferredDevice:    "MOBILE",
		CompletionTendency: tendency,
		AvgCompletionPct:   avgCompletionPct,
		UserInterests:      interests,
		CartSubjects:       cartSubjects,
	}, nil
}

// GetWeeklyReportData returns activity analytics for the last 7 days specifically for the AI Parent Report
func (s *Service) GetWeeklyReportData(ctx context.Context, userID uuid.UUID) (*RecommendationProfile, error) {
	// Get general profile for baseline
	profile, err := s.GetRecommendationProfile(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Filter stats for the last 7 days
	sevenDaysAgo := s.clock.Now().AddDate(0, 0, -7)
	watchSeconds, completedCount, err := s.watchRepo.GetWeeklyWatchStats(ctx, userID, sevenDaysAgo)
	if err != nil {
		return nil, fmt.Errorf("failed to get weekly watch stats: %w", err)
	}

	// For the parent report, we reuse RecommendationProfile but fill it with weekly data
	// AvgSessionDuration -> Total watch time this week
	// AvgCompletionPct -> Lessons completed this week
	profile.AvgSessionDuration = watchSeconds
	profile.AvgCompletionPct = float64(completedCount)

	return profile, nil
}

// ManualRecomputeCourseAnalytics manually triggers course analytics recomputation (for testing/debugging)
func (s *Service) ManualRecomputeCourseAnalytics(ctx context.Context, courseID, userID uuid.UUID) error {
	// Get all lesson progress for this user in this course
	progresses, err := s.watchRepo.GetUserLessonProgressByCourse(ctx, userID, courseID)
	if err != nil {
		return fmt.Errorf("failed to get lesson progress: %w", err)
	}

	// Get total lessons in course
	course, err := s.courseRepo.GetByID(ctx, courseID)
	if err != nil {
		return fmt.Errorf("failed to get course: %w", err)
	}
	if course == nil {
		return ErrCourseNotFound
	}

	now := s.clock.Now()

	// Get or create analytics
	analytics, err := s.watchRepo.GetUserCourseAnalytics(ctx, userID, courseID)
	if err != nil {
		return fmt.Errorf("failed to get course analytics: %w", err)
	}

	if analytics == nil {
		analytics = &watchtime.UserCourseAnalytics{
			ID:        uuid.New(),
			CourseID:  courseID,
			UserID:    userID,
			CreatedAt: now,
		}
	}

	analytics.Recompute(progresses, course.TotalLessons, now)

	if err := s.watchRepo.UpsertUserCourseAnalytics(ctx, analytics); err != nil {
		return fmt.Errorf("failed to upsert course analytics: %w", err)
	}

	return nil
}

// RecordPreviewWatchEvent records a heartbeat for preview/trailer videos (no enrollment required)
func (s *Service) RecordPreviewWatchEvent(ctx context.Context, userID uuid.UUID, input RecordPreviewInput) (*watchtime.UserPreviewProgress, error) {
	// 1. Validate course exists
	course, err := s.courseRepo.GetByID(ctx, input.CourseID)
	if err != nil {
		return nil, fmt.Errorf("failed to get course: %w", err)
	}
	if course == nil {
		return nil, ErrCourseNotFound
	}

	// 2. Check if user is already enrolled (if enrolled, they should use regular heartbeat)
	enrollment, _ := s.enrollmentRepo.GetByCourseAndUser(ctx, input.CourseID, userID)
	if enrollment != nil {
		return nil, errors.New("user is already enrolled, use regular watch heartbeat instead")
	}

	now := s.clock.Now()

	// 3. Insert raw preview watch event
	event := &watchtime.PreviewWatchEvent{
		ID:             uuid.New(),
		CourseID:       input.CourseID,
		UserID:         userID,
		WatchedSeconds: input.WatchedSeconds,
		LastPosition:   input.LastPosition,
		Completed:      input.Completed,
		DeviceType:     input.DeviceType,
		CreatedAt:      now,
	}

	if err := s.watchRepo.CreatePreviewWatchEvent(ctx, event); err != nil {
		return nil, fmt.Errorf("failed to create preview watch event: %w", err)
	}

	// 4. Update aggregated preview progress
	progress, err := s.watchRepo.GetUserPreviewProgress(ctx, userID, input.CourseID)
	if err != nil {
		return nil, fmt.Errorf("failed to get preview progress: %w", err)
	}

	if progress == nil {
		// First time watching this preview
		progress = &watchtime.UserPreviewProgress{
			ID:             uuid.New(),
			CourseID:       input.CourseID,
			UserID:         userID,
			FirstWatchedAt: now,
			CreatedAt:      now,
		}
	}

	// Get preview video duration from course (in seconds)
	// Since Course doesn't have PreviewDuration field, we'll use a default
	// In production, you might want to add this field to the Course model
	// or extract duration from video metadata
	previewDuration := 30 // Default 30 seconds for preview videos
	// TODO: Add PreviewDuration field to Course model or extract from video metadata

	progress.UpdateFromEvent(event, previewDuration, now)

	if err := s.watchRepo.UpsertUserPreviewProgress(ctx, progress); err != nil {
		return nil, fmt.Errorf("failed to upsert preview progress: %w", err)
	}

	// 5. If preview completed, invalidate AI cache to update recommendations
	if input.Completed {
		go s.invalidateAICache(userID.String())
	}

	return progress, nil
}

// GetPreviewProgress returns a user's watch progress on a course preview
func (s *Service) GetPreviewProgress(ctx context.Context, userID, courseID uuid.UUID) (*watchtime.UserPreviewProgress, error) {
	progress, err := s.watchRepo.GetUserPreviewProgress(ctx, userID, courseID)
	if err != nil {
		return nil, fmt.Errorf("failed to get preview progress: %w", err)
	}
	return progress, nil
}
