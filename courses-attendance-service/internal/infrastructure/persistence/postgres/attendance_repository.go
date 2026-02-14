package postgres

import (
	"context"
	"errors"

	"github.com/OmarrGhorab/courses-attendance-service/internal/domain/attendance"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AttendanceSessionRepository implements attendance session data access
type AttendanceSessionRepository struct {
	db *Database
}

func NewAttendanceSessionRepository(db *Database) *AttendanceSessionRepository {
	return &AttendanceSessionRepository{db: db}
}

func (r *AttendanceSessionRepository) Create(ctx context.Context, s *attendance.AttendanceSession) error {
	return r.db.WithContext(ctx).Create(s).Error
}

func (r *AttendanceSessionRepository) GetByLessonID(ctx context.Context, lessonID uuid.UUID) (*attendance.AttendanceSession, error) {
	var s attendance.AttendanceSession
	err := r.db.WithContext(ctx).First(&s, "lesson_id = ?", lessonID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &s, err
}

func (r *AttendanceSessionRepository) Update(ctx context.Context, s *attendance.AttendanceSession) error {
	return r.db.WithContext(ctx).Save(s).Error
}

func (r *AttendanceSessionRepository) GetActive(ctx context.Context) ([]attendance.AttendanceSession, error) {
	var sessions []attendance.AttendanceSession
	err := r.db.WithContext(ctx).Where("is_active = true").Find(&sessions).Error
	return sessions, err
}

// AttendanceQRTokenRepository implements QR token data access
type AttendanceQRTokenRepository struct {
	db *Database
}

func NewAttendanceQRTokenRepository(db *Database) *AttendanceQRTokenRepository {
	return &AttendanceQRTokenRepository{db: db}
}

func (r *AttendanceQRTokenRepository) Create(ctx context.Context, t *attendance.AttendanceQRToken) error {
	return r.db.WithContext(ctx).Create(t).Error
}

func (r *AttendanceQRTokenRepository) GetByLessonAndNonce(ctx context.Context, lessonID uuid.UUID, nonce string) (*attendance.AttendanceQRToken, error) {
	var t attendance.AttendanceQRToken
	err := r.db.WithContext(ctx).First(&t, "lesson_id = ? AND nonce = ?", lessonID, nonce).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &t, err
}

func (r *AttendanceQRTokenRepository) MarkConsumed(ctx context.Context, id uuid.UUID, consumedBy uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&attendance.AttendanceQRToken{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_consumed": true,
			"consumed_by": consumedBy,
			"consumed_at": gorm.Expr("NOW()"),
		}).Error
}

// AttendanceRecordRepository implements attendance record data access
type AttendanceRecordRepository struct {
	db *Database
}

func NewAttendanceRecordRepository(db *Database) *AttendanceRecordRepository {
	return &AttendanceRecordRepository{db: db}
}

func (r *AttendanceRecordRepository) Create(ctx context.Context, rec *attendance.AttendanceRecord) error {
	return r.db.WithContext(ctx).Create(rec).Error
}

func (r *AttendanceRecordRepository) Upsert(ctx context.Context, rec *attendance.AttendanceRecord) error {
	return r.db.WithContext(ctx).Save(rec).Error
}

func (r *AttendanceRecordRepository) GetByID(ctx context.Context, id uuid.UUID) (*attendance.AttendanceRecord, error) {
	var rec attendance.AttendanceRecord
	err := r.db.WithContext(ctx).First(&rec, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &rec, err
}

func (r *AttendanceRecordRepository) GetByLessonAndStudent(ctx context.Context, lessonID, studentID uuid.UUID) (*attendance.AttendanceRecord, error) {
	var rec attendance.AttendanceRecord
	err := r.db.WithContext(ctx).First(&rec, "lesson_id = ? AND student_id = ?", lessonID, studentID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &rec, err
}

func (r *AttendanceRecordRepository) GetByLessonID(ctx context.Context, lessonID uuid.UUID) ([]attendance.AttendanceRecord, error) {
	var records []attendance.AttendanceRecord
	err := r.db.WithContext(ctx).Where("lesson_id = ?", lessonID).Find(&records).Error
	return records, err
}

func (r *AttendanceRecordRepository) GetByStudentID(ctx context.Context, studentID uuid.UUID) ([]attendance.AttendanceRecord, error) {
	var records []attendance.AttendanceRecord
	err := r.db.WithContext(ctx).Where("student_id = ?", studentID).Find(&records).Error
	return records, err
}

func (r *AttendanceRecordRepository) GetByStudentAndLessons(ctx context.Context, studentID uuid.UUID, lessonIDs []uuid.UUID) ([]attendance.AttendanceRecord, error) {
	var records []attendance.AttendanceRecord
	if len(lessonIDs) == 0 {
		return records, nil
	}
	err := r.db.WithContext(ctx).Where("student_id = ? AND lesson_id IN ?", studentID, lessonIDs).Find(&records).Error
	return records, err
}

func (r *AttendanceRecordRepository) Update(ctx context.Context, rec *attendance.AttendanceRecord) error {
	return r.db.WithContext(ctx).Save(rec).Error
}

func (r *AttendanceRecordRepository) BulkCreateAbsent(ctx context.Context, lessonID uuid.UUID, studentIDs []uuid.UUID) error {
	if len(studentIDs) == 0 {
		return nil
	}

	records := make([]attendance.AttendanceRecord, len(studentIDs))
	for i, sid := range studentIDs {
		records[i] = attendance.AttendanceRecord{
			LessonID:  lessonID,
			StudentID: sid,
			Status:    attendance.AttendanceStatusAbsent,
		}
	}
	return r.db.WithContext(ctx).Create(&records).Error
}

func (r *AttendanceRecordRepository) CountByStudentAndCourse(ctx context.Context, studentID, courseID uuid.UUID) (map[attendance.AttendanceStatus]int, error) {
	type result struct {
		Status attendance.AttendanceStatus
		Count  int
	}
	var results []result

	err := r.db.WithContext(ctx).
		Model(&attendance.AttendanceRecord{}).
		Select("attendance_records.status, COUNT(*) as count").
		Joins("JOIN lessons ON lessons.id = attendance_records.lesson_id").
		Where("attendance_records.student_id = ? AND lessons.course_id = ?", studentID, courseID).
		Group("attendance_records.status").
		Scan(&results).Error

	counts := make(map[attendance.AttendanceStatus]int)
	for _, r := range results {
		counts[r.Status] = r.Count
	}
	return counts, err
}
