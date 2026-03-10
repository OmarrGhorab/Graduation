-- Remove free trial support
ALTER TABLE lessons DROP COLUMN IF EXISTS is_free;
ALTER TABLE courses DROP COLUMN IF EXISTS free_trial_lessons;
