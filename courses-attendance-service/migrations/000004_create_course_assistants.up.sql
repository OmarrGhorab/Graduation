-- Migration: 000004_create_course_assistants.up.sql
-- Creates course_assistants junction table

CREATE TABLE IF NOT EXISTS course_assistants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    assistant_id UUID NOT NULL,
    
    -- Permissions
    can_start_lesson BOOLEAN NOT NULL DEFAULT TRUE,
    can_end_lesson BOOLEAN NOT NULL DEFAULT TRUE,
    can_view_attendance BOOLEAN NOT NULL DEFAULT TRUE,
    can_edit_attendance BOOLEAN NOT NULL DEFAULT FALSE,
    
    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Unique constraint
    CONSTRAINT unique_course_assistant UNIQUE (course_id, assistant_id)
);

-- Indexes
CREATE INDEX idx_course_assistants_course_id ON course_assistants(course_id);
CREATE INDEX idx_course_assistants_assistant_id ON course_assistants(assistant_id);
