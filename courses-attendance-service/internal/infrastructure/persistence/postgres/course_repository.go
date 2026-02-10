package postgres

import (
	"context"
	"errors"
	"time"

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
	err := r.db.WithContext(ctx).Preload("Subject").Where("teacher_id = ?", teacherID).Find(&courses).Error
	return courses, err
}

func (r *CourseRepository) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]course.Course, error) {
	var courses []course.Course
	err := r.db.WithContext(ctx).Preload("Subject").Where("id IN ?", ids).Find(&courses).Error
	return courses, err
}

func (r *CourseRepository) GetBySubjectID(ctx context.Context, subjectID uuid.UUID) ([]course.Course, error) {
	var courses []course.Course
	err := r.db.WithContext(ctx).Preload("Subject").Where("subject_id = ?", subjectID).Find(&courses).Error
	return courses, err
}

func (r *CourseRepository) GetAll(ctx context.Context) ([]course.Course, error) {
	var courses []course.Course
	err := r.db.WithContext(ctx).Preload("Subject").Find(&courses).Error
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

// TeacherRatingRepository implements teacher rating data access
type TeacherRatingRepository struct {
	db *Database
}

func NewTeacherRatingRepository(db *Database) *TeacherRatingRepository {
	return &TeacherRatingRepository{db: db}
}

// TeacherAvgRating represents the average rating for a teacher
type TeacherAvgRating struct {
	TeacherID    uuid.UUID `gorm:"column:teacher_id"`
	TotalRatings int       `gorm:"column:total_ratings"`
	AvgRating    float64   `gorm:"column:avg_rating"`
}

func (TeacherAvgRating) TableName() string {
	return "teacher_avg_ratings"
}

// GetTeacherAvgRating gets the average rating for a teacher
func (r *TeacherRatingRepository) GetTeacherAvgRating(ctx context.Context, teacherID uuid.UUID) (*TeacherAvgRating, error) {
	var rating TeacherAvgRating
	err := r.db.WithContext(ctx).Table("teacher_avg_ratings").Where("teacher_id = ?", teacherID).First(&rating).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &rating, err
}

// GetMultipleTeacherAvgRatings gets average ratings for multiple teachers
func (r *TeacherRatingRepository) GetMultipleTeacherAvgRatings(ctx context.Context, teacherIDs []uuid.UUID) (map[uuid.UUID]float64, error) {
	var ratings []TeacherAvgRating
	err := r.db.WithContext(ctx).Table("teacher_avg_ratings").Where("teacher_id IN ?", teacherIDs).Find(&ratings).Error
	if err != nil {
		return nil, err
	}

	result := make(map[uuid.UUID]float64)
	for _, r := range ratings {
		result[r.TeacherID] = r.AvgRating
	}
	return result, nil
}

// CourseRatingRepository implements course rating data access
type CourseRatingRepository struct {
	db *Database
}

func NewCourseRatingRepository(db *Database) *CourseRatingRepository {
	return &CourseRatingRepository{db: db}
}

// CourseRating represents a course rating
type CourseRating struct {
	ID        uuid.UUID `gorm:"column:id;primaryKey"`
	CourseID  uuid.UUID `gorm:"column:course_id"`
	StudentID uuid.UUID `gorm:"column:student_id"`
	Rating    float64   `gorm:"column:rating"`
	Review    string    `gorm:"column:review"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (CourseRating) TableName() string {
	return "course_ratings"
}

// CourseAvgRating represents the average rating for a course
type CourseAvgRating struct {
	CourseID       uuid.UUID `gorm:"column:course_id"`
	TotalRatings   int       `gorm:"column:total_ratings"`
	AvgRating      float64   `gorm:"column:avg_rating"`
	FiveStarCount  int       `gorm:"column:five_star_count"`
	FourStarCount  int       `gorm:"column:four_star_count"`
	ThreeStarCount int       `gorm:"column:three_star_count"`
	TwoStarCount   int       `gorm:"column:two_star_count"`
	OneStarCount   int       `gorm:"column:one_star_count"`
}

func (CourseAvgRating) TableName() string {
	return "course_avg_ratings"
}

// GetCourseAvgRating gets the average rating for a course
func (r *CourseRatingRepository) GetCourseAvgRating(ctx context.Context, courseID uuid.UUID) (*CourseAvgRating, error) {
	var rating CourseAvgRating
	err := r.db.WithContext(ctx).Table("course_avg_ratings").Where("course_id = ?", courseID).First(&rating).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &rating, err
}

// GetMultipleCourseAvgRatings gets average ratings for multiple courses
func (r *CourseRatingRepository) GetMultipleCourseAvgRatings(ctx context.Context, courseIDs []uuid.UUID) (map[uuid.UUID]*CourseAvgRating, error) {
	var ratings []CourseAvgRating
	err := r.db.WithContext(ctx).Table("course_avg_ratings").Where("course_id IN ?", courseIDs).Find(&ratings).Error
	if err != nil {
		return nil, err
	}

	result := make(map[uuid.UUID]*CourseAvgRating)
	for i := range ratings {
		result[ratings[i].CourseID] = &ratings[i]
	}
	return result, nil
}

// GetCourseRatings gets all ratings for a course with pagination
func (r *CourseRatingRepository) GetCourseRatings(ctx context.Context, courseID uuid.UUID, limit, offset int) ([]CourseRating, error) {
	var ratings []CourseRating
	query := r.db.WithContext(ctx).Where("course_id = ?", courseID).Order("created_at DESC")
	
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	
	err := query.Find(&ratings).Error
	return ratings, err
}

// CountCourseRatings counts total ratings for a course
func (r *CourseRatingRepository) CountCourseRatings(ctx context.Context, courseID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&CourseRating{}).Where("course_id = ?", courseID).Count(&count).Error
	return count, err
}
