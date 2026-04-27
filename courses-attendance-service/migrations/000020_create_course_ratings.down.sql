-- Drop trigger
DROP TRIGGER IF EXISTS trigger_refresh_course_ratings ON course_ratings;

-- Drop function
DROP FUNCTION IF EXISTS refresh_course_ratings();

-- Drop materialized view
DROP MATERIALIZED VIEW IF EXISTS course_avg_ratings;

-- Drop indexes
DROP INDEX IF EXISTS idx_course_ratings_rating;
DROP INDEX IF EXISTS idx_course_ratings_student_id;
DROP INDEX IF EXISTS idx_course_ratings_course_id;

-- Drop table
DROP TABLE IF EXISTS course_ratings;
