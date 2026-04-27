-- Migration: 000007_create_attendance.up.sql
-- Creates attendance_sessions, attendance_qr_tokens, and attendance_records tables

-- Attendance sessions (one per live lesson)
CREATE TABLE IF NOT EXISTS attendance_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lesson_id UUID NOT NULL REFERENCES lessons(id) ON DELETE CASCADE,
    
    -- Session timing
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMPTZ,
    
    -- Session state
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    
    -- Stats
    total_scans INTEGER NOT NULL DEFAULT 0,
    
    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- One active session per lesson
    CONSTRAINT unique_lesson_session UNIQUE (lesson_id)
);

-- QR tokens for rotating codes
CREATE TABLE IF NOT EXISTS attendance_qr_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lesson_id UUID NOT NULL REFERENCES lessons(id) ON DELETE CASCADE,
    
    -- Token data
    nonce VARCHAR(64) NOT NULL,
    payload TEXT NOT NULL,
    signature VARCHAR(128) NOT NULL,
    
    -- Validity window
    issued_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    
    -- Usage tracking
    is_consumed BOOLEAN NOT NULL DEFAULT FALSE,
    consumed_by UUID,
    consumed_at TIMESTAMPTZ,
    
    -- Unique nonce per lesson
    CONSTRAINT unique_lesson_nonce UNIQUE (lesson_id, nonce)
);

CREATE INDEX idx_qr_tokens_lesson_id ON attendance_qr_tokens(lesson_id);
CREATE INDEX idx_qr_tokens_expires_at ON attendance_qr_tokens(expires_at);

-- Attendance records (student attendance per lesson)
CREATE TABLE IF NOT EXISTS attendance_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lesson_id UUID NOT NULL REFERENCES lessons(id) ON DELETE CASCADE,
    student_id UUID NOT NULL,
    
    -- Attendance data
    status attendance_status NOT NULL DEFAULT 'ABSENT',
    scanned_at TIMESTAMPTZ,
    
    -- Device/security info
    device_id VARCHAR(255),
    device_fingerprint VARCHAR(255),
    ip_address VARCHAR(45),
    user_agent TEXT,
    
    -- Location data (for offline lessons)
    scan_lat DOUBLE PRECISION,
    scan_lng DOUBLE PRECISION,
    distance_from_location_m DOUBLE PRECISION,
    
    -- QR token used
    qr_token_id UUID REFERENCES attendance_qr_tokens(id),
    
    -- Flags
    is_manual_override BOOLEAN NOT NULL DEFAULT FALSE,
    override_by UUID,
    override_reason TEXT,
    
    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Unique attendance per lesson per student
    CONSTRAINT unique_lesson_attendance UNIQUE (lesson_id, student_id)
);

CREATE INDEX idx_attendance_records_lesson_id ON attendance_records(lesson_id);
CREATE INDEX idx_attendance_records_student_id ON attendance_records(student_id);
CREATE INDEX idx_attendance_records_status ON attendance_records(status);
