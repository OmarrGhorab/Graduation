-- Add delivery_type column to lessons table
ALTER TABLE lessons 
ADD COLUMN delivery_type VARCHAR(20) NOT NULL DEFAULT 'OFFLINE';

-- Add check constraint to ensure valid values
ALTER TABLE lessons 
ADD CONSTRAINT lessons_delivery_type_check 
CHECK (delivery_type IN ('ONLINE', 'OFFLINE'));

-- Add index for filtering by delivery type
CREATE INDEX idx_lessons_delivery_type ON lessons(delivery_type);
