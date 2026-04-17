-- Migration: Add preview video tracking tables
-- Description: Adds tables to track preview video engagement for non-enrolled users

-- Create preview_watch_events table (raw heartbeat events for preview videos)
CREATE TABLE IF NOT EXISTS preview_watch_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    watched_seconds INT NOT NULL DEFAULT 0,
    last_position INT NOT NULL DEFAULT 0,
    completed BOOLEAN NOT NULL DEFAULT FALSE,
    device_type VARCHAR(20) NOT NULL DEFAULT 'DESKTOP',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes for preview_watch_events
CREATE INDEX IF NOT EXISTS idx_preview_watch_events_user_id ON preview_watch_events(user_id);
CREATE INDEX IF NOT EXISTS idx_preview_watch_events_course_id ON preview_watch_events(course_id);
CREATE INDEX IF NOT EXISTS idx_preview_watch_events_created_at ON preview_watch_events(created_at);

-- Create user_preview_progress table (aggregated preview progress per user per course)
CREATE TABLE IF NOT EXISTS user_preview_progress (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    total_watch_time INT NOT NULL DEFAULT 0,
    max_position INT NOT NULL DEFAULT 0,
    watch_count INT NOT NULL DEFAULT 0,
    completion_pct DECIMAL(5,2) NOT NULL DEFAULT 0.00,
    is_completed BOOLEAN NOT NULL DEFAULT FALSE,
    first_watched_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_watched_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, course_id)
);

-- Create indexes for user_preview_progress
CREATE INDEX IF NOT EXISTS idx_user_preview_progress_user_id ON user_preview_progress(user_id);
CREATE INDEX IF NOT EXISTS idx_user_preview_progress_course_id ON user_preview_progress(course_id);
CREATE INDEX IF NOT EXISTS idx_user_preview_progress_last_watched_at ON user_preview_progress(last_watched_at);

-- Add comments
COMMENT ON TABLE preview_watch_events IS 'Raw heartbeat events for preview/trailer video watches (before purchase)';
COMMENT ON TABLE user_preview_progress IS 'Aggregated preview watch progress per user per course';
COMMENT ON COLUMN user_preview_progress.completion_pct IS 'Percentage of preview video watched (0-100)';
COMMENT ON COLUMN user_preview_progress.is_completed IS 'True if user watched 90%+ of preview video';
