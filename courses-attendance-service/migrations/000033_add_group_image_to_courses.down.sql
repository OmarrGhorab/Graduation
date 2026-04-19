-- Migration: Rollback add group_image to courses
-- Added at: 2026-04-18
ALTER TABLE courses DROP COLUMN IF EXISTS group_image;
