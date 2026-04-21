package absence

import (
	"context"
	"errors"

	absenceDomain "github.com/OmarrGhorab/courses-attendance-service/internal/domain/absence"
	attendanceDomain "github.com/OmarrGhorab/courses-attendance-service/internal/domain/attendance"
	"github.com/OmarrGhorab/courses-attendance-service/internal/domain/events"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/authclient"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/clock"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/notificationevents"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/persistence/postgres"
	"github.com/google/uuid"
)

var (
	ErrRequestNotFound      = errors.New("absence request not found")
	ErrUnauthorized         = errors.New("unauthorized to perform this action")
	ErrInvalidStatus        = errors.New("request is not in a pending state")
	ErrAttendanceNotFound   = errors.New("attendance record not found")
	ErrParentNotLinked      = errors.New("user is not linked to this student as a parent")
	ErrDuplicateRequest     = errors.New("an absence request already exists for this lesson")
	ErrRequestAlreadyExists = errors.New("you have already submitted an excuse for this lesson")
)

// Service handles absence request logic
type Service struct {
	absenceRepo *postgres.AbsenceRequestRepository
	recordRepo  *postgres.AttendanceRecordRepository
	lessonRepo  *postgres.LessonRepository
	courseRepo  *postgres.CourseRepository
	authClient  *authclient.Client
	events      *notificationevents.EventDispatcher
	clock       clock.Clock
}

type EnrichedAbsenceRequest struct {
	absenceDomain.AbsenceRequest
	LessonTitle string
	CourseTitle string
	StudentName string
}

func (s *Service) enrichRequests(ctx context.Context, requests []absenceDomain.AbsenceRequest) []EnrichedAbsenceRequest {
	enriched := make([]EnrichedAbsenceRequest, len(requests))
	lessonCache := make(map[uuid.UUID]string)
	courseCache := make(map[uuid.UUID]string)
	studentCache := make(map[uuid.UUID]string)

	for i, r := range requests {
		e := EnrichedAbsenceRequest{AbsenceRequest: r}

		// 1. Get Student Name
		if name, ok := studentCache[r.StudentID]; ok {
			e.StudentName = name
		} else {
			if user, err := s.authClient.GetUserInfo(ctx, r.StudentID.String()); err == nil && user != nil {
				e.StudentName = user.Name
				studentCache[r.StudentID] = user.Name
			}
		}

		// 2. Get Lesson Title
		if title, ok := lessonCache[r.LessonID]; ok {
			e.LessonTitle = title
		} else {
			lesson, _ := s.lessonRepo.GetByID(ctx, r.LessonID)
			if lesson != nil {
				e.LessonTitle = lesson.Title
				lessonCache[r.LessonID] = lesson.Title

				// 3. Get Course Title
				if cTitle, ok := courseCache[lesson.CourseID]; ok {
					e.CourseTitle = cTitle
				} else {
					course, _ := s.courseRepo.GetByID(ctx, lesson.CourseID)
					if course != nil {
						e.CourseTitle = course.Title
						courseCache[lesson.CourseID] = course.Title
					}
				}
			}
		}
		enriched[i] = e
	}
	return enriched
}

func NewService(
	absenceRepo *postgres.AbsenceRequestRepository,
	recordRepo *postgres.AttendanceRecordRepository,
	lessonRepo *postgres.LessonRepository,
	courseRepo *postgres.CourseRepository,
	authClient *authclient.Client,
	events *notificationevents.EventDispatcher,
	clk clock.Clock,
) *Service {
	return &Service{
		absenceRepo: absenceRepo,
		recordRepo:  recordRepo,
		lessonRepo:  lessonRepo,
		courseRepo:  courseRepo,
		authClient:  authClient,
		events:      events,
		clock:       clk,
	}
}

// CreateRequestInput represents data for a new absence request
type CreateRequestInput struct {
	LessonID    uuid.UUID
	StudentID   uuid.UUID
	ReasonType  absenceDomain.AbsenceReasonType
	ReasonText  string
	Attachment  string
	RequestedBy uuid.UUID
}

// CreateRequest creates a new absence request
func (s *Service) CreateRequest(ctx context.Context, input CreateRequestInput) (*EnrichedAbsenceRequest, error) {
	// If the requester is not the student, verify they are a linked parent
	if input.RequestedBy != input.StudentID {
		link, err := s.authClient.VerifyParentLink(ctx, input.RequestedBy.String(), input.StudentID.String())
		if err != nil {
			return nil, err
		}
		if !link.Valid {
			return nil, ErrParentNotLinked
		}
	}

	// Verify lesson exists
	lesson, err := s.lessonRepo.GetByID(ctx, input.LessonID)
	if err != nil {
		return nil, err
	}
	if lesson == nil {
		return nil, errors.New("lesson not found")
	}

	// Check if an absence request already exists for this lesson and student
	existingRequest, err := s.absenceRepo.GetByLessonAndStudent(ctx, input.LessonID, input.StudentID)
	if err != nil {
		return nil, err
	}
	if existingRequest != nil {
		// If there's already a pending or approved request, don't allow duplicate
		if existingRequest.Status == absenceDomain.AbsenceStatusPending || 
		   existingRequest.Status == absenceDomain.AbsenceStatusApproved {
			return nil, ErrRequestAlreadyExists
		}
		// If the previous request was rejected, allow a new one
	}

	// Create request
	req := &absenceDomain.AbsenceRequest{
		ID:            uuid.New(),
		LessonID:      input.LessonID,
		StudentID:     input.StudentID,
		ReasonType:    input.ReasonType,
		ReasonText:    input.ReasonText,
		AttachmentURL: input.Attachment,
		RequestedBy:   input.RequestedBy,
		RequestedAt:   s.clock.Now(),
		Status:        absenceDomain.AbsenceStatusPending,
		CreatedAt:     s.clock.Now(),
		UpdatedAt:     s.clock.Now(),
	}

	// Check if attendance record exists to link it
	record, err := s.recordRepo.GetByLessonAndStudent(ctx, input.LessonID, input.StudentID)
	if err == nil && record != nil {
		req.AttendanceRecordID = &record.ID
	}

	if err := s.absenceRepo.Create(ctx, req); err != nil {
		return nil, err
	}

	s.events.EmitAbsenceRequested(ctx, events.AbsenceRequestedPayload{
		RequestID:   req.ID,
		LessonID:    req.LessonID,
		LessonTitle: lesson.Title,
		CourseID:    lesson.CourseID,
		CourseTitle: "", // Resovled by notification service if needed
		StudentID:   req.StudentID,
		TeacherID:   uuid.Nil, // Resolved by notification service if needed
		Reason:      req.ReasonText,
	})

	// Enrich with titles for the response
	course, _ := s.courseRepo.GetByID(ctx, lesson.CourseID)
	enriched := &EnrichedAbsenceRequest{
		AbsenceRequest: *req,
		LessonTitle:    lesson.Title,
	}
	if course != nil {
		enriched.CourseTitle = course.Title
	}

	return enriched, nil
}

// RespondRequestInput represents a response to an absence request
type RespondRequestInput struct {
	RequestID    uuid.UUID
	RespondedBy  uuid.UUID
	Approve      bool
	ResponseNote string
}

// RespondToRequest allows a parent to approve or reject a request
func (s *Service) RespondToRequest(ctx context.Context, input RespondRequestInput) (*EnrichedAbsenceRequest, error) {
	req, err := s.absenceRepo.GetByID(ctx, input.RequestID)
	if err != nil {
		return nil, err
	}
	if req == nil {
		return nil, ErrRequestNotFound
	}

	if req.Status != absenceDomain.AbsenceStatusPending {
		return nil, ErrInvalidStatus
	}

	// Verify responder authority: either a linked parent or the lesson's teacher
	isAuthorized := false

	// 1. Check if parent
	link, err := s.authClient.VerifyParentLink(ctx, input.RespondedBy.String(), req.StudentID.String())
	if err == nil && link.Valid {
		isAuthorized = true
	}

	// 2. If not parent, check if teacher/owner of the lesson
	if !isAuthorized {
		lesson, err := s.lessonRepo.GetByID(ctx, req.LessonID)
		if err == nil && lesson != nil {
			course, err := s.courseRepo.GetByID(ctx, lesson.CourseID)
			if err == nil && course != nil {
				// Check if this user is the teacher or an authorized assistant
				if course.TeacherID == input.RespondedBy {
					isAuthorized = true
				}
				// TODO: Check assistants if needed
			}
		}
	}

	if !isAuthorized {
		return nil, ErrUnauthorized
	}

	now := s.clock.Now()
	status := absenceDomain.AbsenceStatusRejected
	if input.Approve {
		status = absenceDomain.AbsenceStatusApproved
	}

	req.Status = status
	req.RespondedBy = &input.RespondedBy
	req.RespondedAt = &now
	req.ResponseNote = input.ResponseNote
	req.UpdatedAt = now

	if err := s.absenceRepo.Update(ctx, req); err != nil {
		return nil, err
	}

	// Emit event
	s.events.EmitAbsenceReviewed(ctx, events.AbsenceReviewedPayload{
		RequestID:  req.ID,
		Status:     string(req.Status),
		ReviewerID: input.RespondedBy,
	})

	// If approved, update the attendance record to EXCUSED
	if status == absenceDomain.AbsenceStatusApproved {
		record, err := s.recordRepo.GetByLessonAndStudent(ctx, req.LessonID, req.StudentID)
		if err == nil && record != nil {
			record.Status = attendanceDomain.AttendanceStatusExcused
			record.UpdatedAt = now
			s.recordRepo.Upsert(ctx, record)
		} else if err == nil && record == nil {
			// Create an excused record if it doesn't exist
			newRecord := &attendanceDomain.AttendanceRecord{
				ID:        uuid.New(),
				LessonID:  req.LessonID,
				StudentID: req.StudentID,
				Status:    attendanceDomain.AttendanceStatusExcused,
				CreatedAt: now,
				UpdatedAt: now,
			}
			s.recordRepo.Upsert(ctx, newRecord)
		}
	}

	// Enrich for response
	enriched := s.enrichRequests(ctx, []absenceDomain.AbsenceRequest{*req})
	return &enriched[0], nil
}

// GetStudentRequests returns all absence requests for a student
func (s *Service) GetStudentRequests(ctx context.Context, studentID uuid.UUID) ([]EnrichedAbsenceRequest, error) {
	requests, err := s.absenceRepo.GetByStudentID(ctx, studentID)
	if err != nil {
		return nil, err
	}
	return s.enrichRequests(ctx, requests), nil
}

// GetLessonRequests returns all absence requests for a lesson
func (s *Service) GetLessonRequests(ctx context.Context, lessonID uuid.UUID) ([]EnrichedAbsenceRequest, error) {
	requests, err := s.absenceRepo.GetByLessonID(ctx, lessonID)
	if err != nil {
		return nil, err
	}
	return s.enrichRequests(ctx, requests), nil
}

// GetPendingParentRequests returns all pending requests for students linked to this parent
func (s *Service) GetPendingParentRequests(ctx context.Context, parentID uuid.UUID) ([]EnrichedAbsenceRequest, error) {
	requests, err := s.absenceRepo.GetPendingByParent(ctx, parentID)
	if err != nil {
		return nil, err
	}
	return s.enrichRequests(ctx, requests), nil
}

// GetParentKidsAbsences returns all absence requests (pending, approved, rejected) for all children of a parent
func (s *Service) GetParentKidsAbsences(ctx context.Context, parentID uuid.UUID) ([]EnrichedAbsenceRequest, error) {
	// 1. Get linked kids from auth service
	children, err := s.authClient.GetChildren(ctx, parentID.String())
	if err != nil {
		return nil, err
	}

	if len(children) == 0 {
		return []EnrichedAbsenceRequest{}, nil
	}

	// 2. Extract IDs
	childIDs := make([]uuid.UUID, len(children))
	for i, c := range children {
		uid, err := uuid.Parse(c.ID)
		if err != nil {
			continue
		}
		childIDs[i] = uid
	}

	// 3. Fetch all requests for these kids
	requests, err := s.absenceRepo.GetByStudentIDs(ctx, childIDs)
	if err != nil {
		return nil, err
	}
	return s.enrichRequests(ctx, requests), nil
}
