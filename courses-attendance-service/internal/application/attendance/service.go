package attendance

import (
	"context"
	"errors"
	"time"

	progressApp "github.com/OmarrGhorab/courses-attendance-service/internal/application/progress"
	attendanceDomain "github.com/OmarrGhorab/courses-attendance-service/internal/domain/attendance"
	courseDomain "github.com/OmarrGhorab/courses-attendance-service/internal/domain/course"
	"github.com/OmarrGhorab/courses-attendance-service/internal/domain/events"
	lessonDomain 	"github.com/OmarrGhorab/courses-attendance-service/internal/domain/lesson"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/aiclient"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/authclient"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/cache/redis"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/clock"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/notificationevents"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/persistence/postgres"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/qr"
	"github.com/google/uuid"
)

var (
	ErrLessonNotFound           = errors.New("lesson not found")
	ErrLessonNotLive            = errors.New("lesson is not currently live")
	ErrSessionNotFound          = errors.New("attendance session not found")
	ErrSessionNotActive         = errors.New("attendance session is not active")
	ErrNotEnrolled              = errors.New("student is not enrolled in this course")
	ErrInvalidQRToken           = errors.New("invalid QR token")
	ErrQRTokenExpired           = errors.New("QR token has expired")
	ErrQRNonceConsumed          = errors.New("QR code has already been used")
	ErrOutsideGeofence          = errors.New("you are outside the allowed location")
	ErrRateLimitExceeded        = errors.New("too many scan attempts, please wait")
	ErrScanLockFailed           = errors.New("scan already in progress")
	ErrDeviceVerificationFailed = errors.New("device verification failed")
	ErrEmulatorDetected         = errors.New("emulator detected")
	ErrUnauthorized             = errors.New("unauthorized")
	ErrAlreadyScanned           = errors.New("attendance already recorded")
	ErrSharedDeviceViolation    = errors.New("this device has already been used for attendance in this lesson by another student")
)

// Service handles attendance-related business logic
type Service struct {
	lessonRepo      *postgres.LessonRepository
	courseRepo      *postgres.CourseRepository
	enrollmentRepo  *postgres.EnrollmentRepository
	sessionRepo     *postgres.AttendanceSessionRepository
	qrTokenRepo     *postgres.AttendanceQRTokenRepository
	recordRepo      *postgres.AttendanceRecordRepository
	redisClient     *redis.Client
	authClient      *authclient.Client
	aiClient        *aiclient.Client
	qrGenerator     *qr.Generator
	progressService *progressApp.Service // Injected progress service
	events          *notificationevents.EventDispatcher
	clock           clock.Clock
	rotationSeconds int
	expirySeconds   int
}

type ServiceConfig struct {
	RotationSeconds int
	ExpirySeconds   int
}

func NewService(
	lessonRepo *postgres.LessonRepository,
	courseRepo *postgres.CourseRepository,
	enrollmentRepo *postgres.EnrollmentRepository,
	sessionRepo *postgres.AttendanceSessionRepository,
	qrTokenRepo *postgres.AttendanceQRTokenRepository,
	recordRepo *postgres.AttendanceRecordRepository,
	redisClient *redis.Client,
	authClient *authclient.Client,
	aiClient *aiclient.Client,
	qrGenerator *qr.Generator,
	progressService *progressApp.Service,
	events *notificationevents.EventDispatcher,
	clk clock.Clock,
	cfg ServiceConfig,
) *Service {
	return &Service{
		lessonRepo:      lessonRepo,
		courseRepo:      courseRepo,
		enrollmentRepo:  enrollmentRepo,
		sessionRepo:     sessionRepo,
		qrTokenRepo:     qrTokenRepo,
		recordRepo:      recordRepo,
		redisClient:     redisClient,
		authClient:      authClient,
		aiClient:        aiClient,
		qrGenerator:     qrGenerator,
		progressService: progressService,
		events:          events,
		clock:           clk,
		rotationSeconds: cfg.RotationSeconds,
		expirySeconds:   cfg.ExpirySeconds,
	}
}

// StartAttendanceSession creates a new attendance session for a lesson
func (s *Service) StartAttendanceSession(ctx context.Context, lessonID uuid.UUID) (*attendanceDomain.AttendanceSession, error) {
	// Verify lesson exists and is live
	lesson, err := s.lessonRepo.GetByID(ctx, lessonID)
	if err != nil {
		return nil, err
	}
	if lesson == nil {
		return nil, ErrLessonNotFound
	}
	if lesson.Status != lessonDomain.LessonStatusLive {
		return nil, ErrLessonNotLive
	}

	// Create attendance session
	session := &attendanceDomain.AttendanceSession{
		ID:        uuid.New(),
		LessonID:  lessonID,
		StartedAt: s.clock.Now(),
		IsActive:  true,
		CreatedAt: s.clock.Now(),
	}

	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, err
	}

	// Generate initial QR token
	if _, err := s.generateAndStoreQRToken(ctx, lessonID); err != nil {
		return nil, err
	}

	return session, nil
}

// EndAttendanceSession ends an attendance session and marks absentees
func (s *Service) EndAttendanceSession(ctx context.Context, lessonID uuid.UUID) error {
	session, err := s.sessionRepo.GetByLessonID(ctx, lessonID)
	if err != nil {
		return err
	}
	if session == nil {
		return ErrSessionNotFound
	}

	now := s.clock.Now()
	session.IsActive = false
	session.EndedAt = &now

	if err := s.sessionRepo.Update(ctx, session); err != nil {
		return err
	}

	// Delete active QR token from Redis
	s.redisClient.DeleteActiveQRToken(ctx, lessonID.String())

	// Mark absent students
	if err := s.markAbsentStudents(ctx, lessonID); err != nil {
		return err
	}

	// Emit event
	s.events.Dispatch(ctx, events.TypeAttendanceFinalized, lessonID.String(), uuid.UUID{}, events.AttendanceFinalizedPayload{
		LessonID: lessonID,
	})

	return nil
}

// markAbsentStudents marks all enrolled students without attendance as ABSENT
func (s *Service) markAbsentStudents(ctx context.Context, lessonID uuid.UUID) error {
	lesson, err := s.lessonRepo.GetByID(ctx, lessonID)
	if err != nil {
		return err
	}

	// Get enrolled students
	enrollments, err := s.enrollmentRepo.GetByCourseID(ctx, lesson.CourseID)
	if err != nil {
		return err
	}

	// Get existing attendance records
	existingRecords, err := s.recordRepo.GetByLessonID(ctx, lessonID)
	if err != nil {
		return err
	}

	// Build set of students who already have records
	hasRecord := make(map[uuid.UUID]bool)
	for _, record := range existingRecords {
		hasRecord[record.StudentID] = true
	}

	// Find students without records
	var absentStudentIDs []uuid.UUID
	for _, enrollment := range enrollments {
		if !hasRecord[enrollment.UserID] {
			absentStudentIDs = append(absentStudentIDs, enrollment.UserID)
		}
	}

	// Bulk create absent records
	if len(absentStudentIDs) > 0 {
		if err := s.recordRepo.BulkCreateAbsent(ctx, lessonID, absentStudentIDs); err != nil {
			return err
		}

		// Emit events for each absent student
		course, _ := s.courseRepo.GetByID(ctx, lesson.CourseID)
		for _, studentID := range absentStudentIDs {
			s.events.EmitAttendanceRecorded(ctx, uuid.UUID{}, events.AttendanceRecordedPayload{
				LessonID:    lessonID,
				LessonTitle: lesson.Title,
				CourseID:    lesson.CourseID,
				CourseTitle: course.Title,
				StudentID:   studentID,
				TeacherID:   course.TeacherID,
				Status:      string(attendanceDomain.AttendanceStatusAbsent),
				ScannedAt:   s.clock.Now(),
			})
		}
	}

	return nil
}

// GetCurrentQRToken returns the current active QR token for a lesson
// If no token exists or it has expired, automatically generates a new one
func (s *Service) GetCurrentQRToken(ctx context.Context, lessonID uuid.UUID) (*redis.QRTokenData, error) {
	// Check if session is active
	session, err := s.sessionRepo.GetByLessonID(ctx, lessonID)
	if err != nil {
		return nil, err
	}
	if session == nil || !session.IsActive {
		return nil, ErrSessionNotActive
	}

	// Try to get existing token
	token, err := s.redisClient.GetActiveQRToken(ctx, lessonID.String())
	if err == nil && token != nil {
		// Token exists and is valid
		return token, nil
	}

	// Token doesn't exist or expired, generate a new one automatically
	return s.generateAndStoreQRToken(ctx, lessonID)
}

// RotateQRToken generates a new QR token for a lesson
func (s *Service) RotateQRToken(ctx context.Context, lessonID uuid.UUID) (*redis.QRTokenData, error) {
	return s.generateAndStoreQRToken(ctx, lessonID)
}

func (s *Service) generateAndStoreQRToken(ctx context.Context, lessonID uuid.UUID) (*redis.QRTokenData, error) {
	// Generate new token
	signedToken, err := s.qrGenerator.GenerateToken(lessonID, s.rotationSeconds, s.expirySeconds)
	if err != nil {
		return nil, err
	}

	tokenData := redis.QRTokenData{
		LessonID:  lessonID.String(),
		Nonce:     signedToken.Payload.Nonce,
		Payload:   signedToken.Raw,
		Signature: signedToken.Signature,
		IssuedAt:  signedToken.Payload.IssuedAt,
		ExpiresAt: signedToken.Payload.ExpiresAt,
	}

	// Store in Redis with TTL
	ttl := time.Duration(s.rotationSeconds) * time.Second
	if err := s.redisClient.SetActiveQRToken(ctx, lessonID.String(), tokenData, ttl); err != nil {
		return nil, err
	}

	// Store nonce for validation with extended TTL to support tolerance window
	// Nonce should be valid for: expirySeconds + 2 * toleranceWindow (30s before + 30s after)
	toleranceWindow := 30 * time.Second
	nonceTTL := time.Duration(s.expirySeconds)*time.Second + 2*toleranceWindow
	if err := s.redisClient.SetQRNonce(ctx, lessonID.String(), signedToken.Payload.Nonce, nonceTTL); err != nil {
		return nil, err
	}

	// Store in DB for audit
	dbToken := &attendanceDomain.AttendanceQRToken{
		ID:        uuid.New(),
		LessonID:  lessonID,
		Nonce:     signedToken.Payload.Nonce,
		Payload:   signedToken.Raw,
		Signature: signedToken.Signature,
		IssuedAt:  signedToken.Payload.IssuedAt,
		ExpiresAt: signedToken.Payload.ExpiresAt,
	}
	s.qrTokenRepo.Create(ctx, dbToken)

	return &tokenData, nil
}

// ScanInput represents the input for scanning attendance
type ScanInput struct {
	QRPayload         string
	QRSignature       string
	StudentID         uuid.UUID
	DeviceID          string
	DeviceFingerprint string
	AttestationToken  string
	IP                string
	UserAgent         string
	Latitude          *float64
	Longitude         *float64
	AccessToken       string
}

// ScanResult represents the result of a scan
type ScanResult struct {
	Status    attendanceDomain.AttendanceStatus
	ScannedAt time.Time
	Distance  *float64
	Message   string
}

// ScanAttendance processes a QR code scan for attendance
func (s *Service) ScanAttendance(ctx context.Context, input ScanInput) (*ScanResult, error) {
	serverTime := s.clock.Now()

	// 1. Rate limit check
	allowed, err := s.redisClient.CheckRateLimit(ctx, input.StudentID.String(), 10, time.Minute)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrRateLimitExceeded
	}

	// 2. Validate QR token signature and expiry
	payload, err := s.qrGenerator.ValidateToken(input.QRPayload, input.QRSignature, serverTime)
	if err != nil {
		if errors.Is(err, qr.ErrTokenExpired) {
			return nil, ErrQRTokenExpired
		}
		return nil, ErrInvalidQRToken
	}

	lessonID, err := uuid.Parse(payload.LessonID)
	if err != nil {
		return nil, ErrInvalidQRToken
	}

	// 3. Verify QR nonce exists and is not consumed
	// With tolerance window, we need to check if the nonce is valid
	// The nonce should exist in Redis (not expired) and not be marked as consumed
	nonceExists, err := s.redisClient.CheckQRNonce(ctx, lessonID.String(), payload.Nonce)
	if err != nil {
		return nil, err
	}
	if !nonceExists {
		// Nonce doesn't exist or was already consumed
		return nil, ErrQRNonceConsumed
	}
	
	// Mark nonce as consumed to prevent replay attacks
	// This ensures each QR code can only be scanned once
	if err := s.redisClient.ConsumeQRNonce(ctx, lessonID.String(), payload.Nonce); err != nil {
		return nil, err
	}

	// 4. Verify lesson exists and session is active
	lesson, err := s.lessonRepo.GetByID(ctx, lessonID)
	if err != nil {
		return nil, err
	}
	if lesson == nil {
		return nil, ErrLessonNotFound
	}
	if lesson.Status != lessonDomain.LessonStatusLive {
		return nil, ErrLessonNotLive
	}

	session, err := s.sessionRepo.GetByLessonID(ctx, lessonID)
	if err != nil {
		return nil, err
	}
	if session == nil || !session.IsActive {
		return nil, ErrSessionNotActive
	}

	// 5. Verify student enrollment
	course, err := s.courseRepo.GetByID(ctx, lesson.CourseID)
	if err != nil {
		return nil, err
	}

	enrollment, err := s.enrollmentRepo.GetByCourseAndUser(ctx, lesson.CourseID, input.StudentID)
	if err != nil {
		return nil, err
	}
	if enrollment == nil || !enrollment.IsActive {
		return nil, ErrNotEnrolled
	}

	// 6. Verify device context with auth service (if attestation provided)
	if input.AttestationToken != "" {
		verifyResp, err := s.authClient.VerifyAttendanceContext(ctx, authclient.VerifyContextRequest{
			AccessToken:       input.AccessToken,
			DeviceID:          input.DeviceID,
			DeviceFingerprint: input.DeviceFingerprint,
			AttestationToken:  input.AttestationToken,
			IP:                input.IP,
			UserAgent:         input.UserAgent,
		})
		if err != nil {
			return nil, ErrDeviceVerificationFailed
		}
		if !verifyResp.Valid {
			return nil, ErrUnauthorized
		}
		if verifyResp.EmulatorDetected {
			return nil, ErrEmulatorDetected
		}
	}

	// 7. Geofence validation for offline lessons (DISABLED FOR TESTING)
	var distance *float64
	
	// TODO: Re-enable geofence validation for production
	// For now, skip geofence check to allow testing from anywhere
	_ = distance // Suppress unused variable warning

	// 8. Acquire scan lock
	locked, err := s.redisClient.AcquireScanLock(ctx, lessonID.String(), input.StudentID.String(), 5*time.Second)
	if err != nil {
		return nil, err
	}
	if !locked {
		return nil, ErrScanLockFailed
	}
	defer s.redisClient.ReleaseScanLock(ctx, lessonID.String(), input.StudentID.String())

	// 9. Check for existing attendance record
	existingRecord, err := s.recordRepo.GetByLessonAndStudent(ctx, lessonID, input.StudentID)
	if err != nil {
		return nil, err
	}
	if existingRecord != nil && existingRecord.Status != attendanceDomain.AttendanceStatusAbsent {
		return nil, ErrAlreadyScanned
	}
	
	// 9.5. Check for device reuse (Anti-proxy)
	if input.DeviceID != "" {
		existingDeviceRecord, err := s.recordRepo.GetByLessonAndDevice(ctx, lessonID, input.DeviceID)
		if err != nil {
			return nil, err
		}
		// If a record exists for this device ID but for a DIFFERENT student ID, block it
		if existingDeviceRecord != nil && existingDeviceRecord.StudentID != input.StudentID && existingDeviceRecord.Status != attendanceDomain.AttendanceStatusAbsent {
			// Emit Fraud Detection Event before returning error
			s.events.EmitAttendanceFraudDetected(ctx, events.AttendanceFraudDetectedPayload{
				LessonID:          lessonID,
				LessonTitle:       lesson.Title,
				CourseID:          lesson.CourseID,
				CourseTitle:       course.Title,
				StudentID:         input.StudentID,
				ExistingStudentID: existingDeviceRecord.StudentID,
				DeviceID:          input.DeviceID,
				TeacherID:         course.TeacherID,
				DetectedAt:        serverTime,
			})
			return nil, ErrSharedDeviceViolation
		}
	}

	// 10. Determine attendance status
	status := s.determineAttendanceStatus(serverTime, lesson, course)

	// 11. Create/update attendance record
	record := &attendanceDomain.AttendanceRecord{
		LessonID:              lessonID,
		StudentID:             input.StudentID,
		Status:                status,
		ScannedAt:             &serverTime,
		DeviceID:              input.DeviceID,
		DeviceFingerprint:     input.DeviceFingerprint,
		IPAddress:             input.IP,
		UserAgent:             input.UserAgent,
		ScanLat:               input.Latitude,
		ScanLng:               input.Longitude,
		DistanceFromLocationM: distance,
		CreatedAt:             serverTime,
		UpdatedAt:             serverTime,
	}

	if existingRecord != nil {
		record.ID = existingRecord.ID
	} else {
		record.ID = uuid.New()
	}

	if err := s.recordRepo.Upsert(ctx, record); err != nil {
		return nil, err
	}

	// 12. Update session scan count
	session.TotalScans++
	s.sessionRepo.Update(ctx, session)

	// 13. Trigger progress recomputation (async-ish or simple call)
	// For now, simple call for proof of concept, but usually this is an event
	go s.progressService.RecomputeProgress(context.Background(), lesson.CourseID, input.StudentID)

	// 14. Emit event
	s.events.EmitAttendanceRecorded(ctx, input.StudentID, events.AttendanceRecordedPayload{
		LessonID:    lessonID,
		LessonTitle: lesson.Title,
		CourseID:    lesson.CourseID,
		CourseTitle: course.Title,
		StudentID:   input.StudentID,
		TeacherID:   course.TeacherID,
		Status:      string(status),
		ScannedAt:   serverTime,
	})

	// Trigger AI recommendation refresh for offline/live attendance
	if s.aiClient != nil && (status == attendanceDomain.AttendanceStatusPresent || status == attendanceDomain.AttendanceStatusLate) {
		go s.aiClient.InvalidateRecommendationCache(context.Background(), input.StudentID.String())
	}

	return &ScanResult{
		Status:    status,
		ScannedAt: serverTime,
		Distance:  distance,
		Message:   s.getStatusMessage(status),
	}, nil
}

// ManualOverrideInput represents the input for manual attendance override
type ManualOverrideInput struct {
	LessonID    uuid.UUID
	StudentID   uuid.UUID
	OverriddenBy uuid.UUID
	Status      attendanceDomain.AttendanceStatus
	Reason      string
}

// ManualOverride allows a teacher to manually set/update a student's attendance status
func (s *Service) ManualOverride(ctx context.Context, input ManualOverrideInput) error {
	now := s.clock.Now()

	// 1. Verify lesson exists
	lesson, err := s.lessonRepo.GetByID(ctx, input.LessonID)
	if err != nil {
		return err
	}
	if lesson == nil {
		return ErrLessonNotFound
	}

	// 2. Verify course/teacher authority (optional, but handled by handler middleware mostly)
	course, err := s.courseRepo.GetByID(ctx, lesson.CourseID)
	if err != nil {
		return err
	}

	// 3. Find existing record or create new one
	record, err := s.recordRepo.GetByLessonAndStudent(ctx, input.LessonID, input.StudentID)
	if err != nil {
		return err
	}

	var newRecord *attendanceDomain.AttendanceRecord
	if record != nil {
		record.Status = input.Status
		record.IsManualOverride = true
		record.OverrideBy = &input.OverriddenBy
		record.OverrideReason = input.Reason
		record.UpdatedAt = now
		newRecord = record
	} else {
		newRecord = &attendanceDomain.AttendanceRecord{
			ID:               uuid.New(),
			LessonID:         input.LessonID,
			StudentID:        input.StudentID,
			Status:           input.Status,
			IsManualOverride: true,
			OverrideBy:       &input.OverriddenBy,
			OverrideReason:   input.Reason,
			CreatedAt:        now,
			UpdatedAt:        now,
		}
	}

	if err := s.recordRepo.Upsert(ctx, newRecord); err != nil {
		return err
	}

	// 4. Trigger progress recomputation
	go s.progressService.RecomputeProgress(context.Background(), lesson.CourseID, input.StudentID)

	// 5. Emit event
	s.events.EmitAttendanceRecorded(ctx, input.StudentID, events.AttendanceRecordedPayload{
		LessonID:    input.LessonID,
		LessonTitle: lesson.Title,
		CourseID:    lesson.CourseID,
		CourseTitle: course.Title,
		StudentID:   input.StudentID,
		TeacherID:   course.TeacherID,
		Status:      string(input.Status),
		ScannedAt:   now,
	})

	return nil
}

func (s *Service) determineAttendanceStatus(scanTime time.Time, lesson *lessonDomain.Lesson, course *courseDomain.Course) attendanceDomain.AttendanceStatus {
	if lesson.StartsAt == nil {
		return attendanceDomain.AttendanceStatusPresent
	}

	windowEnd := lesson.StartsAt.Add(time.Duration(course.AttendanceWindowMinutes) * time.Minute)

	if scanTime.Before(windowEnd) || scanTime.Equal(windowEnd) {
		return attendanceDomain.AttendanceStatusPresent
	}

	return attendanceDomain.AttendanceStatusLate
}

func (s *Service) getStatusMessage(status attendanceDomain.AttendanceStatus) string {
	switch status {
	case attendanceDomain.AttendanceStatusPresent:
		return "Attendance recorded: Present"
	case attendanceDomain.AttendanceStatusLate:
		return "Attendance recorded: Late"
	default:
		return "Attendance recorded"
	}
}

// GetLessonAttendance returns all attendance records for a lesson
func (s *Service) GetLessonAttendance(ctx context.Context, lessonID uuid.UUID) ([]attendanceDomain.AttendanceRecord, error) {
	return s.recordRepo.GetByLessonID(ctx, lessonID)
}

// GetStudentAttendance returns all attendance records for a student
func (s *Service) GetStudentAttendance(ctx context.Context, studentID uuid.UUID) ([]attendanceDomain.AttendanceRecord, error) {
	return s.recordRepo.GetByStudentID(ctx, studentID)
}

// GetStudentCourseAnalytics returns analytics for a student in a specific course
func (s *Service) GetStudentCourseAnalytics(ctx context.Context, studentID, courseID uuid.UUID) (*StudentCourseAnalytics, error) {
	// Get enrollment
	enrollment, err := s.enrollmentRepo.GetByCourseAndUser(ctx, courseID, studentID)
	if err != nil || enrollment == nil {
		return nil, ErrNotEnrolled
	}

	// Get all lessons for the course
	lessons, err := s.lessonRepo.GetByCourseID(ctx, courseID)
	if err != nil {
		return nil, err
	}

	// Get attendance records for this student in this course
	lessonIDs := make([]uuid.UUID, len(lessons))
	for i, l := range lessons {
		lessonIDs[i] = l.ID
	}
	
	records, err := s.recordRepo.GetByStudentAndLessons(ctx, studentID, lessonIDs)
	if err != nil {
		return nil, err
	}

	// Calculate attendance rate
	totalLessons := len(lessons)
	completedLessons := 0
	presentCount := 0
	lateCount := 0
	
	recordMap := make(map[uuid.UUID]*attendanceDomain.AttendanceRecord)
	for i := range records {
		recordMap[records[i].LessonID] = &records[i]
		switch records[i].Status {
		case attendanceDomain.AttendanceStatusPresent:
			presentCount++
		case attendanceDomain.AttendanceStatusLate:
			lateCount++
		}
	}

	// Count completed lessons (COMPLETED status)
	for _, l := range lessons {
		if l.Status == "COMPLETED" {
			completedLessons++
		}
	}

	attendanceRate := 0.0
	if totalLessons > 0 {
		attendanceRate = float64(presentCount+lateCount) / float64(totalLessons) * 100
	}

	completionRate := 0.0
	if totalLessons > 0 {
		completionRate = float64(completedLessons) / float64(totalLessons) * 100
	}

	// Calculate weekly attendance (last 7 days)
	weeklyAttendance := s.calculateWeeklyAttendance(records, lessons)

	// Get recent activity (last 10 lessons)
	recentActivity := s.getRecentActivity(lessons, recordMap, 10)

	// Calculate rank (simplified - based on attendance rate)
	rank, totalStudents := s.calculateRank(ctx, courseID, studentID, attendanceRate)

	// Calculate points (simplified)
	points := presentCount*10 + lateCount*5

	return &StudentCourseAnalytics{
		AttendanceRate:   attendanceRate,
		AttendanceChange: 0, // TODO: Calculate change from previous period
		CompletionRate:   completionRate,
		CompletedLessons: completedLessons,
		TotalLessons:     totalLessons,
		WeeklyAttendance: weeklyAttendance,
		Rank:             rank,
		TotalStudents:    totalStudents,
		Points:           points,
		RecentActivity:   recentActivity,
	}, nil
}

func (s *Service) calculateWeeklyAttendance(records []attendanceDomain.AttendanceRecord, lessons []lessonDomain.Lesson) []DailyAttendance {
	// Map lesson IDs to duration
	lessonDuration := make(map[uuid.UUID]int)
	for _, l := range lessons {
		lessonDuration[l.ID] = l.DurationMinutes
	}

	// Calculate hours per day for the last 7 days
	now := time.Now()
	dailyHours := make(map[string]float64)
	days := []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
	
	for _, record := range records {
		if record.ScannedAt != nil {
			daysSince := int(now.Sub(*record.ScannedAt).Hours() / 24)
			if daysSince < 7 {
				dayName := record.ScannedAt.Weekday().String()[:3]
				duration := lessonDuration[record.LessonID]
				dailyHours[dayName] += float64(duration) / 60.0 // Convert to hours
			}
		}
	}

	result := make([]DailyAttendance, 7)
	for i, day := range days {
		result[i] = DailyAttendance{
			Day:   day,
			Hours: dailyHours[day],
		}
	}
	return result
}

func (s *Service) getRecentActivity(lessons []lessonDomain.Lesson, recordMap map[uuid.UUID]*attendanceDomain.AttendanceRecord, limit int) []RecentActivity {
	// Sort lessons by scheduled time (most recent first)
	sortedLessons := make([]lessonDomain.Lesson, len(lessons))
	copy(sortedLessons, lessons)
	
	// Simple bubble sort by scheduled time (descending)
	for i := 0; i < len(sortedLessons)-1; i++ {
		for j := 0; j < len(sortedLessons)-i-1; j++ {
			if sortedLessons[j].ScheduledAt.Before(sortedLessons[j+1].ScheduledAt) {
				sortedLessons[j], sortedLessons[j+1] = sortedLessons[j+1], sortedLessons[j]
			}
		}
	}

	result := []RecentActivity{}
	for _, lesson := range sortedLessons {
		if len(result) >= limit {
			break
		}
		
		status := "ABSENT" // Default if no record
		var scannedAt *time.Time
		
		if record, exists := recordMap[lesson.ID]; exists {
			status = string(record.Status)
			scannedAt = record.ScannedAt
		}

		result = append(result, RecentActivity{
			LessonID:     lesson.ID,
			LessonTitle:  lesson.Title,
			Status:       status,
			ScheduledAt:  lesson.ScheduledAt,
			ScannedAt:    scannedAt,
			DurationMins: lesson.DurationMinutes,
		})
	}
	return result
}

func (s *Service) calculateRank(ctx context.Context, courseID, studentID uuid.UUID, studentRate float64) (int, int) {
	// Get all enrollments for the course
	enrollments, err := s.enrollmentRepo.GetByCourseID(ctx, courseID)
	if err != nil {
		return 0, 0
	}

	totalStudents := len(enrollments)
	rank := 1

	// Count how many students have better attendance
	for _, enrollment := range enrollments {
		if enrollment.UserID == studentID {
			continue
		}
		
		// Get attendance records for this student
		lessons, _ := s.lessonRepo.GetByCourseID(ctx, courseID)
		lessonIDs := make([]uuid.UUID, len(lessons))
		for i, l := range lessons {
			lessonIDs[i] = l.ID
		}
		
		records, _ := s.recordRepo.GetByStudentAndLessons(ctx, enrollment.UserID, lessonIDs)
		
		presentCount := 0
		lateCount := 0
		for _, r := range records {
		switch r.Status {
		case attendanceDomain.AttendanceStatusPresent:
			presentCount++
		case attendanceDomain.AttendanceStatusLate:
			lateCount++
		}
		}
		
		rate := 0.0
		if len(lessons) > 0 {
			rate = float64(presentCount+lateCount) / float64(len(lessons)) * 100
		}
		
		if rate > studentRate {
			rank++
		}
	}

	return rank, totalStudents
}

// StudentCourseAnalytics represents analytics data for a student in a course
type StudentCourseAnalytics struct {
	AttendanceRate   float64
	AttendanceChange float64
	CompletionRate   float64
	CompletedLessons int
	TotalLessons     int
	WeeklyAttendance []DailyAttendance
	Rank             int
	TotalStudents    int
	Points           int
	RecentActivity   []RecentActivity
}

type DailyAttendance struct {
	Day   string
	Hours float64
}

type RecentActivity struct {
	LessonID     uuid.UUID
	LessonTitle  string
	Status       string
	ScheduledAt  time.Time
	ScannedAt    *time.Time
	DurationMins int
}
