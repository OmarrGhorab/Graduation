-- Watch Time Tracking Tables
-- Tracks student video consumption for analytics and AI-powered course recommendations

-- Raw heartbeat events from client (video player pings every ~15s)
CREATE TABLE IF NOT EXISTS lesson_watch_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lesson_id UUID NOT NULL REFERENCES lessons(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    watched_seconds INTEGER NOT NULL DEFAULT 0,
    last_position INTEGER NOT NULL DEFAULT 0,
    completed BOOLEAN NOT NULL DEFAULT FALSE,
    device_type VARCHAR(20) NOT NULL DEFAULT 'DESKTOP',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_watch_events_user_lesson ON lesson_watch_events(user_id, lesson_id);
CREATE INDEX idx_watch_events_created ON lesson_watch_events(created_at);

-- Aggregated per-lesson progress (one row per user-lesson pair)
CREATE TABLE IF NOT EXISTS user_lesson_progress (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lesson_id UUID NOT NULL REFERENCES lessons(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    total_watch_time INTEGER NOT NULL DEFAULT 0,
    max_position INTEGER NOT NULL DEFAULT 0,
    watch_count INTEGER NOT NULL DEFAULT 0,
    completion_pct DECIMAL(5,2) NOT NULL DEFAULT 0.00,
    is_completed BOOLEAN NOT NULL DEFAULT FALSE,
    first_watched_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_watched_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_user_lesson_progress UNIQUE (user_id, lesson_id)
);

CREATE INDEX idx_user_lesson_progress_user ON user_lesson_progress(user_id);
CREATE INDEX idx_user_lesson_progress_lesson ON user_lesson_progress(lesson_id);

-- Aggregated per-course engagement analytics (one row per user-course pair)
CREATE TABLE IF NOT EXISTS user_course_analytics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    total_watch_time INTEGER NOT NULL DEFAULT 0,
    lessons_started INTEGER NOT NULL DEFAULT 0,
    lessons_completed INTEGER NOT NULL DEFAULT 0,
    total_lessons INTEGER NOT NULL DEFAULT 0,
    completion_pct DECIMAL(5,2) NOT NULL DEFAULT 0.00,
    avg_lesson_watch_pct DECIMAL(5,2) NOT NULL DEFAULT 0.00,
    engagement_score DECIMAL(5,2) NOT NULL DEFAULT 0.00,
    last_activity_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_user_course_analytics UNIQUE (user_id, course_id)
);

CREATE INDEX idx_user_course_analytics_user ON user_course_analytics(user_id);
CREATE INDEX idx_user_course_analytics_course ON user_course_analytics(course_id);
CREATE INDEX idx_user_course_analytics_engagement ON user_course_analytics(engagement_score DESC);
