-- Restore normal geofence after testing

-- Restore 50 meter geofence radius
UPDATE courses 
SET geofence_radius_m = 50 
WHERE id = 'c0000011-0000-0000-0000-000000000011';

-- Verify the change
SELECT id, title, delivery_type, geofence_radius_m 
FROM courses 
WHERE id = 'c0000011-0000-0000-0000-000000000011';

-- Expected output:
-- id: c0000011-0000-0000-0000-000000000011
-- title: Mobile App Development Workshop
-- delivery_type: OFFLINE
-- geofence_radius_m: 50
