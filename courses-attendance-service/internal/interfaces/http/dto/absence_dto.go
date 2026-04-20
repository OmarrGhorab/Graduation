package dto

import (
	"time"

	"github.com/OmarrGhorab/courses-attendance-service/internal/domain/absence"
	"github.com/google/uuid"
)

type CreateAbsenceRequest struct {
	LessonID   string `json:"lessonId" validate:"required,uuid"`
	StudentID  string `json:"studentId" validate:"required,uuid"`
	ReasonType string `json:"reasonType" validate:"required,oneof=PARENT_EXCUSE MEDICAL EMERGENCY TECHNICAL PERSONAL"`
	ReasonText string `json:"reasonText"`
	Attachment string `json:"attachment"`
}

type RespondAbsenceRequest struct {
	Approve      bool   `json:"approve"`
	ResponseNote string `json:"responseNote"`
}

type AbsenceRequestResponse struct {
	ID                 uuid.UUID  `json:"id"`
	LessonID           uuid.UUID  `json:"lessonId"`
	StudentID          uuid.UUID  `json:"studentId"`
	ReasonType         string     `json:"reasonType"`
	ReasonText         string     `json:"reasonText"`
	AttachmentURL      string     `json:"attachmentUrl,omitempty"`
	RequestedBy        uuid.UUID  `json:"requestedBy"`
	RequestedAt        time.Time  `json:"requestedAt"`
	Status             string     `json:"status"`
	RespondedBy        *uuid.UUID `json:"respondedBy,omitempty"`
	RespondedAt        *time.Time `json:"respondedAt,omitempty"`
	ResponseNote       string     `json:"responseNote,omitempty"`
	AttendanceRecordID *uuid.UUID `json:"attendanceRecordId,omitempty"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
}

func ToAbsenceRequestResponse(r *absence.AbsenceRequest) AbsenceRequestResponse {
	return AbsenceRequestResponse{
		ID:                 r.ID,
		LessonID:           r.LessonID,
		StudentID:          r.StudentID,
		ReasonType:         string(r.ReasonType),
		ReasonText:         r.ReasonText,
		AttachmentURL:      r.AttachmentURL,
		RequestedBy:        r.RequestedBy,
		RequestedAt:        r.RequestedAt,
		Status:             string(r.Status),
		RespondedBy:        r.RespondedBy,
		RespondedAt:        r.RespondedAt,
		ResponseNote:       r.ResponseNote,
		AttendanceRecordID: r.AttendanceRecordID,
		CreatedAt:          r.CreatedAt,
		UpdatedAt:          r.UpdatedAt,
	}
}
