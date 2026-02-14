=-- Create sample attendance records for student in Advanced Calculus I
-- Student: 3f52987c-a2eb-4978-8eae-99349f83f120
-- Course: c0000001-0000-0000-0000-000000000001

-- Lesson 1: PRESENT
INSERT INTO attendance_records (
    id, lesson_id, student_id, status, scanned_at, created_at, updated_at
) VALUES (
    gen_random_uuid(),
    'b0000001-0000-0000-0000-000000000001',
    '3f52987c-a2eb-4978-8eae-99349f83f120',
    'PRESENT',
    '2026-02-10 09:05:00+00',
    NOW(),
    NOW()
) ON CONFLICT (lesson_id, student_id) DO NOTHING;

-- Lesson 2: PRESENT
INSERT INTO attendance_records (
    id, lesson_id, student_id, status, scanned_at, created_at, updated_at
) VALUES (
    gen_random_uuid(),
    'b0000001-0000-0000-0000-000000000002',
    '3f52987c-a2eb-4978-8eae-99349f83f120',
    'PRESENT',
    '2026-02-12 09:10:00+00',
    NOW(),
    NOW()
) ON CONFLICT (lesson_id, student_id) DO NOTHING;

-- Lesson 3: LATE (currently LIVE)
INSERT INTO attendance_records (
    id, lesson_id, student_id, status, scanned_at, created_at, updated_at
) VALUES (
    gen_random_uuid(),
    'b0000001-0000-0000-0000-000000000003',
    '3f52987c-a2eb-4978-8eae-99349f83f120',
    'LATE',
    '2026-02-15 09:20:00+00',
    NOW(),
    NOW()
) ON CONFLICT (lesson_id, student_id) DO NOTHING;

-- Create progress snapshot
INSERT INTO progress_snapshots (
    course_id, student_id, total_lessons, completed_lessons,
    present_count, late_count, absent_count, excused_count,
    completion_ratio, attendance_ratio, overall_progress, calculated_at
) VALUES (
    'c0000001-0000-0000-0000-000000000001',
    '3f52987c-a2eb-4978-8eae-99349f83f120',
    5,  -- total lessons
    3,  -- completed lessons (2 COMPLETED + 1 LIVE)
    2,  -- present count
    1,  -- late count
    0,  -- absent count
    0,  -- excused count
    0.60,  -- completion ratio (3/5)
    0.60,  -- attendance ratio (3/5)
    60.00,  -- overall progress
    NOW()
) ON CONFLICT (course_id, student_id) DO UPDATE SET
    total_lessons = EXCLUDED.total_lessons,
    completed_lessons = EXCLUDED.completed_lessons,
    present_count = EXCLUDED.present_count,
    late_count = EXCLUDED.late_count,
    absent_count = EXCLUDED.absent_count,
    excused_count = EXCLUDED.excused_count,
    completion_ratio = EXCLUDED.completion_ratio,
    attendance_ratio = EXCLUDED.attendance_ratio,
    overall_progress = EXCLUDED.overall_progress,
    calculated_at = NOW();

-- Verify
SELECT 
    ps.student_id,
    ps.total_lessons,
    ps.completed_lessons,
    ps.present_count,
    ps.late_count,
    ps.absent_count,
    ps.attendance_ratio,
    ps.overall_progress
FROM progress_snapshots ps
WHERE ps.student_id = '3f52987c-a2eb-4978-8eae-99349f83f120'
    AND ps.course_id = 'c0000001-0000-0000-0000-000000000001';
