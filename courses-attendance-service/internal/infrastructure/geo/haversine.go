package geo

import (
	"math"
)

const (
	// Earth's radius in meters
	EarthRadiusM = 6371000.0
)

// Haversine calculates the distance between two points in meters
// using the Haversine formula
func Haversine(lat1, lng1, lat2, lng2 float64) float64 {
	// Convert to radians
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLng := (lng2 - lng1) * math.Pi / 180

	// Haversine formula
	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLng/2)*math.Sin(deltaLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return EarthRadiusM * c
}

// IsWithinGeofence checks if a point is within the geofence radius
func IsWithinGeofence(targetLat, targetLng, checkLat, checkLng float64, radiusM float64) bool {
	distance := Haversine(targetLat, targetLng, checkLat, checkLng)
	return distance <= radiusM
}

// DistanceFromGeofence returns the distance in meters and whether the point is within range
func DistanceFromGeofence(targetLat, targetLng, checkLat, checkLng float64, radiusM float64) (float64, bool) {
	distance := Haversine(targetLat, targetLng, checkLat, checkLng)
	return distance, distance <= radiusM
}
