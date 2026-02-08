package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/OmarrGhorab/courses-attendance-service/internal/domain/lesson"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LessonRepository implements lesson data access
type LessonRepository struct {
	db *Database
}

func NewLessonRepository(db *Database) *LessonRepository {
	return &LessonRepository{db: db}
}

func (r *LessonRepository) Create(ctx context.Context, l *lesson.Lesson) error {
	return r.db.WithContext(ctx).Create(l).Error
}

func (r *LessonRepository) GetByID(ctx context.Context, id uuid.UUID) (*lesson.Lesson, error) {
	var l lesson.Lesson
	err := r.db.WithContext(ctx).First(&l, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &l, err
}

func (r *LessonRepository) Update(ctx context.Context, l *lesson.Lesson) error {
	return r.db.WithContext(ctx).Save(l).Error
}

func (r *LessonRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&lesson.Lesson{}, "id = ?", id).Error
}

func (r *LessonRepository) GetByCourseID(ctx context.Context, courseID uuid.UUID) ([]lesson.Lesson, error) {
	var lessons []lesson.Lesson
	err := r.db.WithContext(ctx).Where("course_id = ?", courseID).Order("lesson_number ASC").Find(&lessons).Error
	return lessons, err
}

func (r *LessonRepository) GetUpcoming(ctx context.Context, courseID uuid.UUID, limit int) ([]lesson.Lesson, error) {
	var lessons []lesson.Lesson
	err := r.db.WithContext(ctx).
		Where("course_id = ? AND status = ?", courseID, lesson.LessonStatusScheduled).
		Order("scheduled_at ASC").
		Limit(limit).
		Find(&lessons).Error
	return lessons, err
}

func (r *LessonRepository) GetByStatus(ctx context.Context, status lesson.LessonStatus) ([]lesson.Lesson, error) {
	var lessons []lesson.Lesson
	err := r.db.WithContext(ctx).Where("status = ?", status).Find(&lessons).Error
	return lessons, err
}

func (r *LessonRepository) CountByCourse(ctx context.Context, courseID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&lesson.Lesson{}).Where("course_id = ?", courseID).Count(&count).Error
	return count, err
}

func (r *LessonRepository) CountCompletedByCourse(ctx context.Context, courseID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&lesson.Lesson{}).Where("course_id = ? AND status = ?", courseID, lesson.LessonStatusCompleted).Count(&count).Error
	return count, err
}

func (r *LessonRepository) GetByCoursesAndTimeRange(ctx context.Context, courseIDs []uuid.UUID, start, end time.Time) ([]lesson.Lesson, error) {
	var lessons []lesson.Lesson
	err := r.db.WithContext(ctx).
		Where("course_id IN ? AND scheduled_at BETWEEN ? AND ?", courseIDs, start, end).
		Order("scheduled_at ASC").
		Find(&lessons).Error
	return lessons, err
}
