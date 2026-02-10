package dto

import (
	"time"

	"github.com/google/uuid"
)

// CourseReviewResponse represents a single review for a course
type CourseReviewResponse struct {
	ID              uuid.UUID `json:"id"`
	StudentID       uuid.UUID `json:"studentId"`
	StudentName     string    `json:"studentName"`
	StudentProfile  string    `json:"studentProfile,omitempty"`
	Rating          float64   `json:"rating"`
	Review          string    `json:"review,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// CourseReviewsResponse represents the full response for course reviews
type CourseReviewsResponse struct {
	CourseID         uuid.UUID              `json:"courseId"`
	CourseTitle      string                 `json:"courseTitle"`
	AverageRating    float64                `json:"averageRating"`
	TotalRatings     int                    `json:"totalRatings"`
	RatingBreakdown  RatingBreakdown        `json:"ratingBreakdown"`
	Reviews          []CourseReviewResponse `json:"reviews"`
	Pagination       PaginationInfo         `json:"pagination"`
}

// RatingBreakdown shows distribution of ratings
type RatingBreakdown struct {
	FiveStars  int `json:"fiveStars"`
	FourStars  int `json:"fourStars"`
	ThreeStars int `json:"threeStars"`
	TwoStars   int `json:"twoStars"`
	OneStar    int `json:"oneStar"`
}

// PaginationInfo contains pagination metadata
type PaginationInfo struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalPages int   `json:"totalPages"`
	TotalItems int64 `json:"totalItems"`
}
