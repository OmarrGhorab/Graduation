-- Migration: 000003_create_courses.up.sql
-- Creates courses table

CREATE TABLE IF NOT EXISTS courses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Basic info
    title VARCHAR(255) NOT NULL,
    description TEXT,
    subject_id UUID NOT NULL REFERENCES subjects(id) ON DELETE RESTRICT,
    
    -- Owner (teacher)
    teacher_id UUID NOT NULL,
    
    -- Delivery and location
    delivery_type delivery_type NOT NULL DEFAULT 'OFFLINE',
    location_name VARCHAR(255),
    location_lat DOUBLE PRECISION,
    location_lng DOUBLE PRECISION,
    geofence_radius_m INTEGER DEFAULT 100,
    
    -- Scheduling
    total_lessons INTEGER NOT NULL DEFAULT 0,
    attendance_window_minutes INTEGER NOT NULL DEFAULT 15,
    
    -- Pricing
    price DECIMAL(10, 2) DEFAULT 0.00,
    is_paid BOOLEAN NOT NULL DEFAULT FALSE,
    
    -- Status
    status course_status NOT NULL DEFAULT 'ACTIVE',
    
    -- Progress settings
    attendance_weight DECIMAL(3, 2) NOT NULL DEFAULT 0.30,
    
    -- Timestamps (UTC)
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_courses_teacher_id ON courses(teacher_id);
CREATE INDEX idx_courses_subject_id ON courses(subject_id);
CREATE INDEX idx_courses_status ON courses(status);
