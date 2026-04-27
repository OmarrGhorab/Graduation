-- Migration: 000032_add_lesson_thumbnail.up.sql
ALTER TABLE lessons ADD COLUMN IF NOT EXISTS thumbnail_url TEXT;
