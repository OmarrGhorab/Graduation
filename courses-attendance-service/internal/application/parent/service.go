package parent

import (
	"context"
	"errors"

	attendanceApp "github.com/OmarrGhorab/courses-attendance-service/internal/application/attendance"
	progressApp "github.com/OmarrGhorab/courses-attendance-service/internal/application/progress"
	attendanceDomain "github.com/OmarrGhorab/courses-attendance-service/internal/domain/attendance"
	lessonDomain "github.com/OmarrGhorab/courses-attendance-service/internal/domain/lesson"
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

type ParentAttendanceRecord struct {
	attendanceDomain.AttendanceRecord
	LessonTitle string
	CourseTitle string
}

type Service struct {
	authClient        *authclient.Client
	progressService   *progressApp.Service
	attendanceService *attendanceApp.Service
	enrollmentRepo    *postgres.EnrollmentRepository
	courseRepo        *postgres.CourseRepository
	lessonRepo        *postgres.LessonRepository
	recordRepo        *postgres.AttendanceRecordRepository
}

func NewService(
	authClient *authclient.Client,
	progressService *progressApp.Service,
	attendanceService *attendanceApp.Service,
	enrollmentRepo *postgres.EnrollmentRepository,
	courseRepo *postgres.CourseRepository,
	lessonRepo *postgres.LessonRepository,
	recordRepo *postgres.AttendanceRecordRepository,
) *Service {
	return &Service{
		authClient:        authClient,
		progressService:   progressService,
		attendanceService: attendanceService,
		enrollmentRepo:    enrollmentRepo,
		courseRepo:        courseRepo,
		lessonRepo:        lessonRepo,
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
			TeacherName: "Unknown Teacher",
		}

		// Fetch teacher name from Auth service
		if teacher, err := s.authClient.GetUserInfo(ctx, course.TeacherID.String()); err == nil && teacher != nil {
			cp.TeacherName = teacher.Name
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

func (s *Service) GetChildAttendanceHistory(ctx context.Context, parentID, studentID uuid.UUID) ([]ParentAttendanceRecord, error) {
	// 1. Verify link
	link, err := s.authClient.VerifyParentLink(ctx, parentID.String(), studentID.String())
	if err != nil || !link.Valid {
		return nil, ErrParentNotLinked
	}

	// 2. Fetch records
	records, err := s.recordRepo.GetByStudentID(ctx, studentID)
	if err != nil {
		return nil, err
	}

	// 3. Enrich with titles
	enriched := make([]ParentAttendanceRecord, len(records))
	lessonCache := make(map[uuid.UUID]*lessonDomain.Lesson)
	courseCache := make(map[uuid.UUID]string)

	for i, r := range records {
		item := ParentAttendanceRecord{AttendanceRecord: r}
		
		// Lookup Lesson
		lesson, ok := lessonCache[r.LessonID]
		if !ok {
			lesson, _ = s.lessonRepo.GetByID(ctx, r.LessonID)
			lessonCache[r.LessonID] = lesson
		}

		if lesson != nil {
			item.LessonTitle = lesson.Title
			
			// Lookup Course
			cTitle, ok := courseCache[lesson.CourseID]
			if !ok {
				course, _ := s.courseRepo.GetByID(ctx, lesson.CourseID)
				if course != nil {
					cTitle = course.Title
					courseCache[lesson.CourseID] = cTitle
				}
			}
			item.CourseTitle = cTitle
		}
		
		enriched[i] = item
	}

	return enriched, nil
}
