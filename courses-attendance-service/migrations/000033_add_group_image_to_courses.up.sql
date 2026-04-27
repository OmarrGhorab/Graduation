-- Migration: Add group_image to courses
-- Added at: 2026-04-17
ALTER TABLE courses ADD COLUMN IF NOT EXISTS group_image TEXT;
