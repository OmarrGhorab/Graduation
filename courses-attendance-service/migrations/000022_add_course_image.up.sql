-- Add course_image column to courses table
ALTER TABLE courses ADD COLUMN IF NOT EXISTS course_image TEXT;

-- Add comment
COMMENT ON COLUMN courses.course_image IS 'URL to the course cover/thumbnail image';
