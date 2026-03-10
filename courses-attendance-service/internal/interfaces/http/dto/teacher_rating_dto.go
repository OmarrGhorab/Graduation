package dto

import "github.com/google/uuid"

// TeacherRatingResponse represents a teacher's rating information
type TeacherRatingResponse struct {
	TeacherID         uuid.UUID `json:"teacherId"`
	TeacherName       string    `json:"teacherName,omitempty"`
	TeacherProfileImg string    `json:"teacherProfileImg,omitempty"`
	AverageRating     float64   `json:"averageRating"`
	TotalRatings      int       `json:"totalRatings"`
}
