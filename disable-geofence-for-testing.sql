-- Temporarily disable geofence for testing with emulator
-- This allows scanning from any location (USA emulator, Egypt, anywhere)

-- Increase geofence radius to cover the whole world (20,000 km)
UPDATE courses 
SET geofence_radius_m = 20000000 
WHERE id = 'c0000011-0000-0000-0000-000000000011';

-- Verify the change
SELECT id, title, delivery_type, geofence_radius_m 
FROM courses 
WHERE id = 'c0000011-0000-0000-0000-000000000011';

-- Expected output:
-- id: c0000011-0000-0000-0000-000000000011
-- title: Mobile App Development Workshop
-- delivery_type: OFFLINE
-- geofence_radius_m: 20000000

-- NOTE: This is for TESTING ONLY!
-- To restore normal geofence (50 meters), run:
-- UPDATE courses SET geofence_radius_m = 50 WHERE id = 'c0000011-0000-0000-0000-000000000011';
