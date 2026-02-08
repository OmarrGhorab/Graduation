-- Migration: 000001_init_enums.down.sql
-- Drops all enum types

DROP TYPE IF EXISTS absence_status;
DROP TYPE IF EXISTS absence_reason_type;
DROP TYPE IF EXISTS attendance_status;
DROP TYPE IF EXISTS lesson_status;
DROP TYPE IF EXISTS course_status;
DROP TYPE IF EXISTS delivery_type;
