-- Migration: 000027_convert_to_normal_views.up.sql
-- Converts performance-focused materialized views to real-time regular views

BEGIN;

-- Drop existing triggers and functions (no longer needed for regular views)
DROP TRIGGER IF EXISTS trigger_refresh_course_ratings ON course_ratings;
DROP TRIGGER IF EXISTS trigger_refresh_teacher_ratings ON teacher_ratings;
DROP FUNCTION IF EXISTS refresh_course_ratings();
DROP FUNCTION IF EXISTS refresh_teacher_ratings();

-- Drop materialized views
DROP MATERIALIZED VIEW IF EXISTS course_avg_ratings;
DROP MATERIALIZED VIEW IF EXISTS teacher_avg_ratings;

-- Create live view for course ratings
CREATE OR REPLACE VIEW course_avg_ratings AS
SELECT 
    course_id,
    COUNT(*)::INTEGER as total_ratings,
    AVG(rating)::DOUBLE PRECISION as avg_rating,
    COUNT(CASE WHEN rating = 5.0 THEN 1 END)::INTEGER as five_star_count,
    COUNT(CASE WHEN rating >= 4.0 AND rating < 5.0 THEN 1 END)::INTEGER as four_star_count,
    COUNT(CASE WHEN rating >= 3.0 AND rating < 4.0 THEN 1 END)::INTEGER as three_star_count,
    COUNT(CASE WHEN rating >= 2.0 AND rating < 3.0 THEN 1 END)::INTEGER as two_star_count,
    COUNT(CASE WHEN rating < 2.0 THEN 1 END)::INTEGER as one_star_count,
    MAX(updated_at) as last_updated
FROM course_ratings
GROUP BY course_id;

-- Create live view for teacher ratings
CREATE OR REPLACE VIEW teacher_avg_ratings AS
SELECT 
    teacher_id,
    COUNT(*)::INTEGER as total_ratings,
    AVG(rating)::DOUBLE PRECISION as avg_rating,
    MAX(updated_at) as last_updated
FROM teacher_ratings
GROUP BY teacher_id;

COMMIT;
