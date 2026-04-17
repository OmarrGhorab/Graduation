-- Migration: 000032_add_lesson_thumbnail.down.sql
ALTER TABLE lessons DROP COLUMN IF EXISTS thumbnail_url;
