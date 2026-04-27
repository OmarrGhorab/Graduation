-- Drop trigger
DROP TRIGGER IF EXISTS trigger_refresh_teacher_ratings ON teacher_ratings;

-- Drop function
DROP FUNCTION IF EXISTS refresh_teacher_ratings();

-- Drop materialized view
DROP MATERIALIZED VIEW IF EXISTS teacher_avg_ratings;

-- Drop indexes
DROP INDEX IF EXISTS idx_teacher_ratings_student_id;
DROP INDEX IF EXISTS idx_teacher_ratings_course_id;
DROP INDEX IF EXISTS idx_teacher_ratings_teacher_id;

-- Drop table
DROP TABLE IF EXISTS teacher_ratings;
