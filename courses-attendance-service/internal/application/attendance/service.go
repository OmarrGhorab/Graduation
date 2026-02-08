package attendance

import (
	"context"
	"errors"
	"time"

	progressApp "github.com/OmarrGhorab/courses-attendance-service/internal/application/progress"
	attendanceDomain "github.com/OmarrGhorab/courses-attendance-service/internal/domain/attendance"
	courseDomain "github.com/OmarrGhorab/courses-attendance-service/internal/domain/course"
	"github.com/OmarrGhorab/courses-attendance-service/internal/domain/events"
	lessonDomain "github.com/OmarrGhorab/courses-attendance-service/internal/domain/lesson"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/authclient"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/cache/redis"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/clock"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/geo"
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
	}

	return nil
}

// GetCurrentQRToken returns the current active QR token for a lesson
func (s *Service) GetCurrentQRToken(ctx context.Context, lessonID uuid.UUID) (*redis.QRTokenData, error) {
	// Check if session is active
	session, err := s.sessionRepo.GetByLessonID(ctx, lessonID)
	if err != nil {
		return nil, err
	}
	if session == nil || !session.IsActive {
		return nil, ErrSessionNotActive
	}

	return s.redisClient.GetActiveQRToken(ctx, lessonID.String())
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

	// Also store nonce for validation
	nonceTTL := time.Duration(s.expirySeconds) * time.Second
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
	nonceExists, err := s.redisClient.CheckQRNonce(ctx, lessonID.String(), payload.Nonce)
	if err != nil {
		return nil, err
	}
	if !nonceExists {
		return nil, ErrQRNonceConsumed
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

	// 7. Geofence validation for offline courses
	var distance *float64
	if course.DeliveryType == courseDomain.DeliveryTypeOffline {
		if input.Latitude == nil || input.Longitude == nil {
			return nil, ErrOutsideGeofence
		}

		// Get location (lesson override or course default)
		targetLat := course.LocationLat
		targetLng := course.LocationLng
		radiusM := float64(course.GeofenceRadiusM)

		if lesson.LocationLat != nil {
			targetLat = lesson.LocationLat
		}
		if lesson.LocationLng != nil {
			targetLng = lesson.LocationLng
		}
		if lesson.GeofenceRadiusM != nil {
			radiusM = float64(*lesson.GeofenceRadiusM)
		}

		if targetLat == nil || targetLng == nil {
			return nil, ErrOutsideGeofence
		}

		dist, withinRange := geo.DistanceFromGeofence(*targetLat, *targetLng, *input.Latitude, *input.Longitude, radiusM)
		distance = &dist

		if !withinRange {
			return nil, ErrOutsideGeofence
		}
	}

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
		LessonID:  lessonID,
		StudentID: input.StudentID,
		Status:    string(status),
		ScannedAt: serverTime,
	})

	return &ScanResult{
		Status:    status,
		ScannedAt: serverTime,
		Distance:  distance,
		Message:   s.getStatusMessage(status),
	}, nil
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
