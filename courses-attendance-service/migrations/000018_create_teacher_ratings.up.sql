-- Create teacher_ratings table
CREATE TABLE IF NOT EXISTS teacher_ratings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    teacher_id UUID NOT NULL,
    student_id UUID NOT NULL,
    course_id UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    rating DECIMAL(2,1) NOT NULL CHECK (rating >= 0.0 AND rating <= 5.0),
    review TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(teacher_id, student_id, course_id)
);

-- Create index for faster lookups
CREATE INDEX idx_teacher_ratings_teacher_id ON teacher_ratings(teacher_id);
CREATE INDEX idx_teacher_ratings_course_id ON teacher_ratings(course_id);
CREATE INDEX idx_teacher_ratings_student_id ON teacher_ratings(student_id);

-- Create materialized view for teacher average ratings (for performance)
CREATE MATERIALIZED VIEW teacher_avg_ratings AS
SELECT 
    teacher_id,
    COUNT(*) as total_ratings,
    AVG(rating) as avg_rating,
    MAX(updated_at) as last_updated
FROM teacher_ratings
GROUP BY teacher_id;

-- Create index on materialized view
CREATE UNIQUE INDEX idx_teacher_avg_ratings_teacher_id ON teacher_avg_ratings(teacher_id);

-- Function to refresh materialized view
CREATE OR REPLACE FUNCTION refresh_teacher_ratings()
RETURNS TRIGGER AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY teacher_avg_ratings;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Trigger to auto-refresh on rating changes
CREATE TRIGGER trigger_refresh_teacher_ratings
AFTER INSERT OR UPDATE OR DELETE ON teacher_ratings
FOR EACH STATEMENT
EXECUTE FUNCTION refresh_teacher_ratings();
