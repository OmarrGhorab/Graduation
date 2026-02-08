-- Migration: 000002_create_subjects.up.sql
-- Creates subjects table (predefined course categories)

CREATE TABLE IF NOT EXISTS subjects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    icon VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed common subjects
INSERT INTO subjects (name, description, icon) VALUES
    ('Mathematics', 'Math and arithmetic courses', 'calculator'),
    ('Physics', 'Physics and mechanics courses', 'atom'),
    ('Chemistry', 'Chemistry and lab courses', 'flask'),
    ('Biology', 'Biology and life sciences', 'dna'),
    ('Arabic', 'Arabic language courses', 'book'),
    ('English', 'English language courses', 'globe'),
    ('French', 'French language courses', 'flag'),
    ('History', 'History and social studies', 'landmark'),
    ('Geography', 'Geography courses', 'map'),
    ('Computer Science', 'Programming and CS courses', 'laptop')
ON CONFLICT (name) DO NOTHING;
