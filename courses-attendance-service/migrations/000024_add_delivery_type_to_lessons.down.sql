-- Remove index
DROP INDEX IF EXISTS idx_lessons_delivery_type;

-- Remove check constraint
ALTER TABLE lessons 
DROP CONSTRAINT IF EXISTS lessons_delivery_type_check;

-- Remove delivery_type column
ALTER TABLE lessons 
DROP COLUMN IF EXISTS delivery_type;
