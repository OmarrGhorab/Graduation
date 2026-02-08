package postgres

import (
	"context"
	"errors"

	"github.com/OmarrGhorab/courses-attendance-service/internal/domain/course"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CourseRepository implements course data access
type CourseRepository struct {
	db *Database
}

func NewCourseRepository(db *Database) *CourseRepository {
	return &CourseRepository{db: db}
}

func (r *CourseRepository) Create(ctx context.Context, c *course.Course) error {
	return r.db.WithContext(ctx).Create(c).Error
}

func (r *CourseRepository) GetByID(ctx context.Context, id uuid.UUID) (*course.Course, error) {
	var c course.Course
	err := r.db.WithContext(ctx).Preload("Subject").First(&c, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &c, err
}

func (r *CourseRepository) Update(ctx context.Context, c *course.Course) error {
	return r.db.WithContext(ctx).Save(c).Error
}

func (r *CourseRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&course.Course{}, "id = ?", id).Error
}

func (r *CourseRepository) GetByTeacherID(ctx context.Context, teacherID uuid.UUID) ([]course.Course, error) {
	var courses []course.Course
	err := r.db.WithContext(ctx).Where("teacher_id = ?", teacherID).Find(&courses).Error
	return courses, err
}

func (r *CourseRepository) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]course.Course, error) {
	var courses []course.Course
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&courses).Error
	return courses, err
}

func (r *CourseRepository) GetBySubjectID(ctx context.Context, subjectID uuid.UUID) ([]course.Course, error) {
	var courses []course.Course
	err := r.db.WithContext(ctx).Where("subject_id = ?", subjectID).Find(&courses).Error
	return courses, err
}

// SubjectRepository implements subject data access
type SubjectRepository struct {
	db *Database
}

func NewSubjectRepository(db *Database) *SubjectRepository {
	return &SubjectRepository{db: db}
}

func (r *SubjectRepository) GetAll(ctx context.Context) ([]course.Subject, error) {
	var subjects []course.Subject
	err := r.db.WithContext(ctx).Find(&subjects).Error
	return subjects, err
}

func (r *SubjectRepository) GetByID(ctx context.Context, id uuid.UUID) (*course.Subject, error) {
	var s course.Subject
	err := r.db.WithContext(ctx).First(&s, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &s, err
}

// EnrollmentRepository implements enrollment data access
type EnrollmentRepository struct {
	db *Database
}

func NewEnrollmentRepository(db *Database) *EnrollmentRepository {
	return &EnrollmentRepository{db: db}
}

func (r *EnrollmentRepository) Create(ctx context.Context, e *course.Enrollment) error {
	return r.db.WithContext(ctx).Create(e).Error
}

func (r *EnrollmentRepository) GetByID(ctx context.Context, id uuid.UUID) (*course.Enrollment, error) {
	var e course.Enrollment
	err := r.db.WithContext(ctx).First(&e, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &e, err
}

func (r *EnrollmentRepository) GetByCourseAndUser(ctx context.Context, courseID, userID uuid.UUID) (*course.Enrollment, error) {
	var e course.Enrollment
	err := r.db.WithContext(ctx).First(&e, "course_id = ? AND user_id = ?", courseID, userID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &e, err
}

func (r *EnrollmentRepository) GetByCourseID(ctx context.Context, courseID uuid.UUID) ([]course.Enrollment, error) {
	var enrollments []course.Enrollment
	err := r.db.WithContext(ctx).Where("course_id = ? AND is_active = true", courseID).Find(&enrollments).Error
	return enrollments, err
}

func (r *EnrollmentRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]course.Enrollment, error) {
	var enrollments []course.Enrollment
	err := r.db.WithContext(ctx).Where("user_id = ? AND is_active = true", userID).Find(&enrollments).Error
	return enrollments, err
}

func (r *EnrollmentRepository) Update(ctx context.Context, e *course.Enrollment) error {
	return r.db.WithContext(ctx).Save(e).Error
}

// CourseAssistantRepository implements course assistant data access
type CourseAssistantRepository struct {
	db *Database
}

func NewCourseAssistantRepository(db *Database) *CourseAssistantRepository {
	return &CourseAssistantRepository{db: db}
}

func (r *CourseAssistantRepository) Create(ctx context.Context, ca *course.CourseAssistant) error {
	return r.db.WithContext(ctx).Create(ca).Error
}

func (r *CourseAssistantRepository) GetByCourseID(ctx context.Context, courseID uuid.UUID) ([]course.CourseAssistant, error) {
	var assistants []course.CourseAssistant
	err := r.db.WithContext(ctx).Where("course_id = ?", courseID).Find(&assistants).Error
	return assistants, err
}

func (r *CourseAssistantRepository) GetByCourseAndAssistant(ctx context.Context, courseID, assistantID uuid.UUID) (*course.CourseAssistant, error) {
	var ca course.CourseAssistant
	err := r.db.WithContext(ctx).First(&ca, "course_id = ? AND assistant_id = ?", courseID, assistantID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &ca, err
}

func (r *CourseAssistantRepository) Delete(ctx context.Context, courseID, assistantID uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&course.CourseAssistant{}, "course_id = ? AND assistant_id = ?", courseID, assistantID).Error
}
