-- Remove course_image column from courses table
ALTER TABLE courses DROP COLUMN IF EXISTS course_image;
