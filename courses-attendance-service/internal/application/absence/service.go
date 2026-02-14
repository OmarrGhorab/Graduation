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
	authClient  *authclient.Client
	events      *notificationevents.EventDispatcher
	clock       clock.Clock
}

func NewService(
	absenceRepo *postgres.AbsenceRequestRepository,
	recordRepo *postgres.AttendanceRecordRepository,
	lessonRepo *postgres.LessonRepository,
	authClient *authclient.Client,
	events *notificationevents.EventDispatcher,
	clk clock.Clock,
) *Service {
	return &Service{
		absenceRepo: absenceRepo,
		recordRepo:  recordRepo,
		lessonRepo:  lessonRepo,
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
func (s *Service) CreateRequest(ctx context.Context, input CreateRequestInput) (*absenceDomain.AbsenceRequest, error) {
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

	// Emit event
	s.events.EmitAbsenceRequested(ctx, events.AbsenceRequestedPayload{
		RequestID: req.ID,
		LessonID:  req.LessonID,
		StudentID: req.StudentID,
		Reason:    req.ReasonText,
	})

	return req, nil
}

// RespondRequestInput represents a response to an absence request
type RespondRequestInput struct {
	RequestID    uuid.UUID
	RespondedBy  uuid.UUID
	Approve      bool
	ResponseNote string
}

// RespondToRequest allows a parent to approve or reject a request
func (s *Service) RespondToRequest(ctx context.Context, input RespondRequestInput) (*absenceDomain.AbsenceRequest, error) {
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

	// Verify the responder is a linked parent
	link, err := s.authClient.VerifyParentLink(ctx, input.RespondedBy.String(), req.StudentID.String())
	if err != nil {
		return nil, err
	}
	if !link.Valid {
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

	return req, nil
}

// GetStudentRequests returns all absence requests for a student
func (s *Service) GetStudentRequests(ctx context.Context, studentID uuid.UUID) ([]absenceDomain.AbsenceRequest, error) {
	return s.absenceRepo.GetByStudentID(ctx, studentID)
}

// GetLessonRequests returns all absence requests for a lesson
func (s *Service) GetLessonRequests(ctx context.Context, lessonID uuid.UUID) ([]absenceDomain.AbsenceRequest, error) {
	return s.absenceRepo.GetByLessonID(ctx, lessonID)
}

// GetPendingParentRequests returns all pending requests for students linked to this parent
func (s *Service) GetPendingParentRequests(ctx context.Context, parentID uuid.UUID) ([]absenceDomain.AbsenceRequest, error) {
	// In a real scenario, we might need a list of child IDs first
	// For now, let's use the repository method which might need to be refined if we don't have child list
	return s.absenceRepo.GetPendingByParent(ctx, parentID)
}
