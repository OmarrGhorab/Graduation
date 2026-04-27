-- Migration: 000009_create_progress_snapshots.up.sql
-- Creates progress_snapshots table for student progress tracking

CREATE TABLE IF NOT EXISTS progress_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    student_id UUID NOT NULL,
    
    -- Progress metrics
    total_lessons INTEGER NOT NULL DEFAULT 0,
    completed_lessons INTEGER NOT NULL DEFAULT 0,
    
    -- Attendance breakdown
    present_count INTEGER NOT NULL DEFAULT 0,
    late_count INTEGER NOT NULL DEFAULT 0,
    absent_count INTEGER NOT NULL DEFAULT 0,
    excused_count INTEGER NOT NULL DEFAULT 0,
    
    -- Calculated scores
    completion_ratio DECIMAL(5, 4) NOT NULL DEFAULT 0.0000,
    attendance_ratio DECIMAL(5, 4) NOT NULL DEFAULT 0.0000,
    overall_progress DECIMAL(5, 2) NOT NULL DEFAULT 0.00,
    
    -- Snapshot timing
    calculated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Unique progress per student per course
    CONSTRAINT unique_course_student_progress UNIQUE (course_id, student_id)
);

CREATE INDEX idx_progress_snapshots_course_id ON progress_snapshots(course_id);
CREATE INDEX idx_progress_snapshots_student_id ON progress_snapshots(student_id);
