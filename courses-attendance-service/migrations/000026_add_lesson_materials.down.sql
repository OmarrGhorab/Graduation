-- Remove lesson materials support
ALTER TABLE lessons DROP COLUMN IF EXISTS duration;
ALTER TABLE lessons DROP COLUMN IF EXISTS materials_url;
ALTER TABLE lessons DROP COLUMN IF EXISTS video_public_id;
ALTER TABLE lessons DROP COLUMN IF EXISTS video_url;
