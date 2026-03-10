-- Add lesson materials support for ONLINE courses
ALTER TABLE lessons ADD COLUMN IF NOT EXISTS video_url TEXT;
ALTER TABLE lessons ADD COLUMN IF NOT EXISTS video_public_id VARCHAR(255);
ALTER TABLE lessons ADD COLUMN IF NOT EXISTS materials_url TEXT;
ALTER TABLE lessons ADD COLUMN IF NOT EXISTS duration INTEGER;

-- Comments for documentation
COMMENT ON COLUMN lessons.video_url IS 'Cloudinary video URL for ONLINE lessons';
COMMENT ON COLUMN lessons.video_public_id IS 'Cloudinary public ID for video management';
COMMENT ON COLUMN lessons.materials_url IS 'URL to additional materials (PDFs, slides, etc.)';
COMMENT ON COLUMN lessons.duration IS 'Video duration in seconds';
