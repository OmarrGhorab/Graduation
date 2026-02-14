-- Create course_ratings table (separate from teacher ratings)
CREATE TABLE IF NOT EXISTS course_ratings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    student_id UUID NOT NULL,
    rating DECIMAL(2,1) NOT NULL CHECK (rating >= 0.0 AND rating <= 5.0),
    review TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(course_id, student_id)
);

-- Create indexes for faster lookups
CREATE INDEX idx_course_ratings_course_id ON course_ratings(course_id);
CREATE INDEX idx_course_ratings_student_id ON course_ratings(student_id);
CREATE INDEX idx_course_ratings_rating ON course_ratings(rating);

-- Create materialized view for course average ratings
CREATE MATERIALIZED VIEW course_avg_ratings AS
SELECT 
    course_id,
    COUNT(*) as total_ratings,
    AVG(rating) as avg_rating,
    COUNT(CASE WHEN rating = 5.0 THEN 1 END) as five_star_count,
    COUNT(CASE WHEN rating >= 4.0 AND rating < 5.0 THEN 1 END) as four_star_count,
    COUNT(CASE WHEN rating >= 3.0 AND rating < 4.0 THEN 1 END) as three_star_count,
    COUNT(CASE WHEN rating >= 2.0 AND rating < 3.0 THEN 1 END) as two_star_count,
    COUNT(CASE WHEN rating < 2.0 THEN 1 END) as one_star_count,
    MAX(updated_at) as last_updated
FROM course_ratings
GROUP BY course_id;

-- Create unique index on materialized view
CREATE UNIQUE INDEX idx_course_avg_ratings_course_id ON course_avg_ratings(course_id);

-- Function to refresh course ratings materialized view
CREATE OR REPLACE FUNCTION refresh_course_ratings()
RETURNS TRIGGER AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY course_avg_ratings;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Trigger to auto-refresh on rating changes
CREATE TRIGGER trigger_refresh_course_ratings
AFTER INSERT OR UPDATE OR DELETE ON course_ratings
FOR EACH STATEMENT
EXECUTE FUNCTION refresh_course_ratings();
