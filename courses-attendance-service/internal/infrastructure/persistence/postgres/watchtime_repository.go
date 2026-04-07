package postgres

import (
	"context"
	"errors"

	"github.com/OmarrGhorab/courses-attendance-service/internal/domain/watchtime"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// WatchTimeRepository implements watch time data access
type WatchTimeRepository struct {
	db *Database
}

func NewWatchTimeRepository(db *Database) *WatchTimeRepository {
	return &WatchTimeRepository{db: db}
}

// --- Watch Events ---

// CreateWatchEvent inserts a raw heartbeat event
func (r *WatchTimeRepository) CreateWatchEvent(ctx context.Context, event *watchtime.LessonWatchEvent) error {
	return r.db.WithContext(ctx).Create(event).Error
}

// --- User Lesson Progress ---

// GetUserLessonProgress retrieves aggregated progress for a user on a lesson
func (r *WatchTimeRepository) GetUserLessonProgress(ctx context.Context, userID, lessonID uuid.UUID) (*watchtime.UserLessonProgress, error) {
	var p watchtime.UserLessonProgress
	err := r.db.WithContext(ctx).First(&p, "user_id = ? AND lesson_id = ?", userID, lessonID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &p, err
}

// UpsertUserLessonProgress creates or updates user lesson progress
func (r *WatchTimeRepository) UpsertUserLessonProgress(ctx context.Context, p *watchtime.UserLessonProgress) error {
	return r.db.WithContext(ctx).Save(p).Error
}

// GetUserLessonProgressByCourse retrieves all lesson progress for a user in a specific course
func (r *WatchTimeRepository) GetUserLessonProgressByCourse(ctx context.Context, userID, courseID uuid.UUID) ([]watchtime.UserLessonProgress, error) {
	var progresses []watchtime.UserLessonProgress
	err := r.db.WithContext(ctx).
		Table("user_lesson_progress").
		Joins("INNER JOIN lessons ON lessons.id = user_lesson_progress.lesson_id").
		Where("user_lesson_progress.user_id = ? AND lessons.course_id = ?", userID, courseID).
		Find(&progresses).Error
	return progresses, err
}

// --- User Course Analytics ---

// GetUserCourseAnalytics retrieves course analytics for a user
func (r *WatchTimeRepository) GetUserCourseAnalytics(ctx context.Context, userID, courseID uuid.UUID) (*watchtime.UserCourseAnalytics, error) {
	var a watchtime.UserCourseAnalytics
	err := r.db.WithContext(ctx).First(&a, "user_id = ? AND course_id = ?", userID, courseID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &a, err
}

// UpsertUserCourseAnalytics creates or updates user course analytics
func (r *WatchTimeRepository) UpsertUserCourseAnalytics(ctx context.Context, a *watchtime.UserCourseAnalytics) error {
	return r.db.WithContext(ctx).Save(a).Error
}

// GetUserAllCourseAnalytics retrieves all course analytics for a user (for dashboard/recommendations)
func (r *WatchTimeRepository) GetUserAllCourseAnalytics(ctx context.Context, userID uuid.UUID) ([]watchtime.UserCourseAnalytics, error) {
	var analytics []watchtime.UserCourseAnalytics
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("last_activity_at DESC").
		Find(&analytics).Error
	return analytics, err
}

// GetCourseEngagementLeaderboard returns top students by engagement score for a course
func (r *WatchTimeRepository) GetCourseEngagementLeaderboard(ctx context.Context, courseID uuid.UUID, limit int) ([]watchtime.UserCourseAnalytics, error) {
	var analytics []watchtime.UserCourseAnalytics
	query := r.db.WithContext(ctx).
		Where("course_id = ?", courseID).
		Order("engagement_score DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&analytics).Error
	return analytics, err
}

// SubjectWatchStat holds aggregated watch data per subject for recommendation
type SubjectWatchStat struct {
	SubjectID       uuid.UUID `gorm:"column:subject_id"`
	SubjectName     string    `gorm:"column:subject_name"`
	TotalWatchTime  int       `gorm:"column:total_watch_time"`
	CoursesWatched  int       `gorm:"column:courses_watched"`
	AvgEngagement   float64   `gorm:"column:avg_engagement"`
	AvgCompletionPct float64  `gorm:"column:avg_completion_pct"`
}

// GetMostWatchedSubjects returns a user's most-engaged subjects (for recommendations)
func (r *WatchTimeRepository) GetMostWatchedSubjects(ctx context.Context, userID uuid.UUID, limit int) ([]SubjectWatchStat, error) {
	var stats []SubjectWatchStat
	err := r.db.WithContext(ctx).
		Table("user_course_analytics uca").
		Select(`
			c.subject_id,
			s.name as subject_name,
			SUM(uca.total_watch_time) as total_watch_time,
			COUNT(DISTINCT uca.course_id) as courses_watched,
			AVG(uca.engagement_score) as avg_engagement,
			AVG(uca.completion_pct) as avg_completion_pct
		`).
		Joins("INNER JOIN courses c ON c.id = uca.course_id").
		Joins("INNER JOIN subjects s ON s.id = c.subject_id").
		Where("uca.user_id = ?", userID).
		Group("c.subject_id, s.name").
		Order("avg_engagement DESC").
		Limit(limit).
		Scan(&stats).Error
	return stats, err
}

// GetAllLessonProgressForUser returns all lesson progress across all courses for a user
func (r *WatchTimeRepository) GetAllLessonProgressForUser(ctx context.Context, userID uuid.UUID) ([]watchtime.UserLessonProgress, error) {
	var progresses []watchtime.UserLessonProgress
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("last_watched_at DESC").
		Find(&progresses).Error
	return progresses, err
}
