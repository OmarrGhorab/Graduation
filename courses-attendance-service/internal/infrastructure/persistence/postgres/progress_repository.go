package postgres

import (
	"context"
	"errors"

	"github.com/OmarrGhorab/courses-attendance-service/internal/domain/progress"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ProgressSnapshotRepository implements progress snapshot data access
type ProgressSnapshotRepository struct {
	db *Database
}

func NewProgressSnapshotRepository(db *Database) *ProgressSnapshotRepository {
	return &ProgressSnapshotRepository{db: db}
}

func (r *ProgressSnapshotRepository) Upsert(ctx context.Context, p *progress.ProgressSnapshot) error {
	return r.db.WithContext(ctx).Save(p).Error
}

func (r *ProgressSnapshotRepository) GetByCourseAndStudent(ctx context.Context, courseID, studentID uuid.UUID) (*progress.ProgressSnapshot, error) {
	var p progress.ProgressSnapshot
	err := r.db.WithContext(ctx).First(&p, "course_id = ? AND student_id = ?", courseID, studentID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &p, err
}

func (r *ProgressSnapshotRepository) GetByCourseID(ctx context.Context, courseID uuid.UUID) ([]progress.ProgressSnapshot, error) {
	var snapshots []progress.ProgressSnapshot
	err := r.db.WithContext(ctx).Where("course_id = ?", courseID).Find(&snapshots).Error
	return snapshots, err
}

func (r *ProgressSnapshotRepository) GetByStudentID(ctx context.Context, studentID uuid.UUID) ([]progress.ProgressSnapshot, error) {
	var snapshots []progress.ProgressSnapshot
	err := r.db.WithContext(ctx).Where("student_id = ?", studentID).Find(&snapshots).Error
	return snapshots, err
}
