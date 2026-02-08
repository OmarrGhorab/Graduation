-- Migration: 000005_create_lessons.up.sql
-- Creates lessons table

CREATE TABLE IF NOT EXISTS lessons (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    
    -- Lesson info
    title VARCHAR(255) NOT NULL,
    description TEXT,
    lesson_number INTEGER NOT NULL,
    
    -- Scheduling (all UTC)
    scheduled_at TIMESTAMPTZ NOT NULL,
    starts_at TIMESTAMPTZ,
    ends_at TIMESTAMPTZ,
    duration_minutes INTEGER NOT NULL DEFAULT 60,
    
    -- Status
    status lesson_status NOT NULL DEFAULT 'SCHEDULED',
    
    -- Location override (optional, inherits from course if null)
    location_name VARCHAR(255),
    location_lat DOUBLE PRECISION,
    location_lng DOUBLE PRECISION,
    geofence_radius_m INTEGER,
    
    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_lessons_course_id ON lessons(course_id);
CREATE INDEX idx_lessons_status ON lessons(status);
CREATE INDEX idx_lessons_scheduled_at ON lessons(scheduled_at);
CREATE UNIQUE INDEX idx_lessons_course_lesson_number ON lessons(course_id, lesson_number);
