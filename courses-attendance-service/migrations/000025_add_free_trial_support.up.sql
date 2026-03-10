-- Add free trial support to courses
ALTER TABLE courses ADD COLUMN IF NOT EXISTS free_trial_lessons INTEGER NOT NULL DEFAULT 0;

-- Add is_free flag to lessons
ALTER TABLE lessons ADD COLUMN IF NOT EXISTS is_free BOOLEAN NOT NULL DEFAULT false;

-- Comment for documentation
COMMENT ON COLUMN courses.free_trial_lessons IS 'Number of free lessons for trial (0 = all paid or all free based on is_paid)';
COMMENT ON COLUMN lessons.is_free IS 'True if this lesson is free (for trial purposes)';
