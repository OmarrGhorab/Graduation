-- Migration: 000008_create_absence_requests.up.sql
-- Creates absence_requests table for parent approval flow

CREATE TABLE IF NOT EXISTS absence_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lesson_id UUID NOT NULL REFERENCES lessons(id) ON DELETE CASCADE,
    student_id UUID NOT NULL,
    
    -- Request details
    reason_type absence_reason_type NOT NULL,
    reason_text TEXT,
    
    -- Attachments (medical certificate, etc.)
    attachment_url TEXT,
    
    -- Parent/requester info
    requested_by UUID NOT NULL,
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Response
    status absence_status NOT NULL DEFAULT 'PENDING',
    responded_by UUID,
    responded_at TIMESTAMPTZ,
    response_note TEXT,
    
    -- Link to attendance record
    attendance_record_id UUID REFERENCES attendance_records(id),
    
    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_absence_requests_lesson_id ON absence_requests(lesson_id);
CREATE INDEX idx_absence_requests_student_id ON absence_requests(student_id);
CREATE INDEX idx_absence_requests_status ON absence_requests(status);
CREATE INDEX idx_absence_requests_requested_by ON absence_requests(requested_by);
