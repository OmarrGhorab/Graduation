package parent

import (
	"context"
	"errors"

	attendanceApp "github.com/OmarrGhorab/courses-attendance-service/internal/application/attendance"
	progressApp "github.com/OmarrGhorab/courses-attendance-service/internal/application/progress"
	attendanceDomain "github.com/OmarrGhorab/courses-attendance-service/internal/domain/attendance"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/authclient"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/persistence/postgres"
	"github.com/google/uuid"
)

var (
	ErrParentNotLinked = errors.New("user is not linked to this student as a parent")
)

type ChildSummary struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	ProfileImg string    `json:"profileImg"`
	Email      string    `json:"email"`
	Relation   string    `json:"relation"`
}

type ChildDetailedProgress struct {
	Child ChildSummary `json:"child"`
	Courses []CourseProgress `json:"courses"`
}

type CourseProgress struct {
	CourseID       uuid.UUID `json:"courseId"`
	CourseTitle    string    `json:"courseTitle"`
	TeacherName    string    `json:"teacherName"`
	OverallProgress float64   `json:"overallProgress"`
	AttendanceRate  float64   `json:"attendanceRate"`
	PresentCount    int       `json:"presentCount"`
	AbsentCount     int       `json:"absentCount"`
	LateCount       int       `json:"lateCount"`
	ExcusedCount    int       `json:"excusedCount"`
	TotalLessons    int       `json:"totalLessons"`
}

type Service struct {
	authClient        *authclient.Client
	progressService   *progressApp.Service
	attendanceService *attendanceApp.Service
	enrollmentRepo    *postgres.EnrollmentRepository
	courseRepo        *postgres.CourseRepository
	recordRepo        *postgres.AttendanceRecordRepository
}

func NewService(
	authClient *authclient.Client,
	progressService *progressApp.Service,
	attendanceService *attendanceApp.Service,
	enrollmentRepo *postgres.EnrollmentRepository,
	courseRepo *postgres.CourseRepository,
	recordRepo *postgres.AttendanceRecordRepository,
) *Service {
	return &Service{
		authClient:        authClient,
		progressService:   progressService,
		attendanceService: attendanceService,
		enrollmentRepo:    enrollmentRepo,
		courseRepo:        courseRepo,
		recordRepo:        recordRepo,
	}
}

func (s *Service) GetChildren(ctx context.Context, parentID uuid.UUID) ([]ChildSummary, error) {
	children, err := s.authClient.GetChildren(ctx, parentID.String())
	if err != nil {
		return nil, err
	}

	result := make([]ChildSummary, len(children))
	for i, c := range children {
		uid, _ := uuid.Parse(c.ID)
		result[i] = ChildSummary{
			ID:         uid,
			Name:       c.Name,
			ProfileImg: c.ProfileImg,
			Email:      c.Email,
			Relation:   c.Relation,
		}
	}
	return result, nil
}

func (s *Service) GetChildDetailedProgress(ctx context.Context, parentID, studentID uuid.UUID) (*ChildDetailedProgress, error) {
	// 1. Verify link
	link, err := s.authClient.VerifyParentLink(ctx, parentID.String(), studentID.String())
	if err != nil || !link.Valid {
		return nil, ErrParentNotLinked
	}

	// 2. Get child info
	childInfo, err := s.authClient.GetUserInfo(ctx, studentID.String())
	if err != nil {
		return nil, err
	}

	// 3. Get enrolled courses
	enrollments, err := s.enrollmentRepo.GetByUserID(ctx, studentID)
	if err != nil {
		return nil, err
	}

	courseProgressList := []CourseProgress{}
	for _, enrollment := range enrollments {
		course, err := s.courseRepo.GetByID(ctx, enrollment.CourseID)
		if err != nil || course == nil {
			continue
		}

		// Get progress snapshot
		snapshot, err := s.progressService.GetStudentProgress(ctx, course.ID, studentID)
		if err != nil || snapshot == nil {
			// If no snapshot, try to recompute once or just use zeros
			snapshot, _ = s.progressService.RecomputeProgress(ctx, course.ID, studentID)
		}

		cp := CourseProgress{
			CourseID:    course.ID,
			CourseTitle: course.Title,
			TeacherName: "", // In a real scenario, we'd fetch teacher's name
		}

		if snapshot != nil {
			cp.OverallProgress = snapshot.OverallProgress
			cp.AttendanceRate = snapshot.AttendanceRatio * 100 // Corrected field
			cp.PresentCount = snapshot.PresentCount
			cp.AbsentCount = snapshot.AbsentCount
			cp.LateCount = snapshot.LateCount
			cp.ExcusedCount = snapshot.ExcusedCount
			cp.TotalLessons = snapshot.TotalLessons
		}

		courseProgressList = append(courseProgressList, cp)
	}

	return &ChildDetailedProgress{
		Child: ChildSummary{
			ID:         studentID,
			Name:       childInfo.Name,
			ProfileImg: childInfo.ProfileImg,
			Email:      childInfo.Email,
			Relation:   link.Relation,
		},
		Courses: courseProgressList,
	}, nil
}

func (s *Service) GetChildAttendanceHistory(ctx context.Context, parentID, studentID uuid.UUID) ([]attendanceDomain.AttendanceRecord, error) {
	// 1. Verify link
	link, err := s.authClient.VerifyParentLink(ctx, parentID.String(), studentID.String())
	if err != nil || !link.Valid {
		return nil, ErrParentNotLinked
	}

	// 2. Fetch history
	return s.recordRepo.GetByStudentID(ctx, studentID)
}
