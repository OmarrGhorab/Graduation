package postgres

import (
	"context"
	"errors"

	"github.com/OmarrGhorab/courses-attendance-service/internal/domain/absence"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AbsenceRequestRepository implements absence request data access
type AbsenceRequestRepository struct {
	db *Database
}

func NewAbsenceRequestRepository(db *Database) *AbsenceRequestRepository {
	return &AbsenceRequestRepository{db: db}
}

func (r *AbsenceRequestRepository) Create(ctx context.Context, req *absence.AbsenceRequest) error {
	return r.db.WithContext(ctx).Create(req).Error
}

func (r *AbsenceRequestRepository) GetByID(ctx context.Context, id uuid.UUID) (*absence.AbsenceRequest, error) {
	var req absence.AbsenceRequest
	err := r.db.WithContext(ctx).First(&req, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &req, err
}

func (r *AbsenceRequestRepository) Update(ctx context.Context, req *absence.AbsenceRequest) error {
	return r.db.WithContext(ctx).Save(req).Error
}

func (r *AbsenceRequestRepository) GetByStudentID(ctx context.Context, studentID uuid.UUID) ([]absence.AbsenceRequest, error) {
	var requests []absence.AbsenceRequest
	err := r.db.WithContext(ctx).Where("student_id = ?", studentID).Order("created_at DESC").Find(&requests).Error
	return requests, err
}

func (r *AbsenceRequestRepository) GetByLessonID(ctx context.Context, lessonID uuid.UUID) ([]absence.AbsenceRequest, error) {
	var requests []absence.AbsenceRequest
	err := r.db.WithContext(ctx).Where("lesson_id = ?", lessonID).Find(&requests).Error
	return requests, err
}

func (r *AbsenceRequestRepository) GetPendingByParent(ctx context.Context, parentID uuid.UUID) ([]absence.AbsenceRequest, error) {
	var requests []absence.AbsenceRequest
	err := r.db.WithContext(ctx).Where("requested_by = ? AND status = ?", parentID, absence.AbsenceStatusPending).Find(&requests).Error
	return requests, err
}

func (r *AbsenceRequestRepository) GetPendingForStudent(ctx context.Context, studentID uuid.UUID) ([]absence.AbsenceRequest, error) {
	var requests []absence.AbsenceRequest
	err := r.db.WithContext(ctx).Where("student_id = ? AND status = ?", studentID, absence.AbsenceStatusPending).Find(&requests).Error
	return requests, err
}
