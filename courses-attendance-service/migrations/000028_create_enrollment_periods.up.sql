-- Migration: create enrollment_periods table
CREATE TABLE IF NOT EXISTS enrollment_periods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    enrollment_id UUID NOT NULL REFERENCES enrollments(id) ON DELETE CASCADE,
    period_key VARCHAR(10) NOT NULL, -- Format: YYYY-MM (e.g., 2026-04)
    is_paid BOOLEAN NOT NULL DEFAULT FALSE,
    paid_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(enrollment_id, period_key)
);

CREATE INDEX IF NOT EXISTS idx_enrollment_periods_enrollment_id ON enrollment_periods(enrollment_id);
CREATE INDEX IF NOT EXISTS idx_enrollment_periods_period_key ON enrollment_periods(period_key);
