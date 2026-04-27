-- Migration: 000001_init_enums.up.sql
-- Creates all enum types for the courses-attendance-service

-- Delivery type for courses (online vs offline)
CREATE TYPE delivery_type AS ENUM ('ONLINE', 'OFFLINE');

-- Course status lifecycle
CREATE TYPE course_status AS ENUM ('ACTIVE', 'PAUSED', 'ARCHIVED');

-- Lesson status lifecycle
CREATE TYPE lesson_status AS ENUM ('SCHEDULED', 'LIVE', 'COMPLETED', 'CANCELED');

-- Attendance status for students
CREATE TYPE attendance_status AS ENUM ('PRESENT', 'LATE', 'ABSENT', 'EXCUSED');

-- Absence reason category
CREATE TYPE absence_reason_type AS ENUM ('PARENT_EXCUSE', 'MEDICAL', 'EMERGENCY');

-- Absence request status
CREATE TYPE absence_status AS ENUM ('PENDING', 'APPROVED', 'REJECTED');
