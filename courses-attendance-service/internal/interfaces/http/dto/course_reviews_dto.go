package dto

import (
	"time"

	"github.com/google/uuid"
)

// CreateCourseReviewRequest represents a request to create a course review
type CreateCourseReviewRequest struct {
	Rating float64 `json:"rating" validate:"required,min=1,max=5"`
	Review string  `json:"review" validate:"required,min=10,max=1000"`
}

// UpdateCourseReviewRequest represents a request to update a course review
type UpdateCourseReviewRequest struct {
	Rating float64 `json:"rating" validate:"required,min=1,max=5"`
	Review string  `json:"review" validate:"required,min=10,max=1000"`
}

// CourseReviewResponse represents a single course review
type CourseReviewResponse struct {
	ID              uuid.UUID `json:"id"`
	StudentID       uuid.UUID `json:"studentId"`
	StudentName     string    `json:"studentName"`
	StudentUsername string    `json:"studentUsername"`
	StudentProfile  string    `json:"studentProfile,omitempty"`
	Rating          float64   `json:"rating"`
	Review          string    `json:"review"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}


// RatingBreakdown represents the distribution of ratings
type RatingBreakdown struct {
	FiveStars  int `json:"fiveStars"`
	FourStars  int `json:"fourStars"`
	ThreeStars int `json:"threeStars"`
	TwoStars   int `json:"twoStars"`
	OneStar    int `json:"oneStar"`
}

// PaginationInfo represents pagination metadata
type PaginationInfo struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalItems int64 `json:"totalItems"`
	TotalPages int   `json:"totalPages"`
}

// CourseReviewsResponse represents the response for course reviews
type CourseReviewsResponse struct {
	CourseID        uuid.UUID              `json:"courseId"`
	CourseTitle     string                 `json:"courseTitle"`
	AverageRating   float64                `json:"averageRating"`
	TotalRatings    int                    `json:"totalRatings"`
	RatingBreakdown RatingBreakdown        `json:"ratingBreakdown"`
	Reviews         []CourseReviewResponse `json:"reviews"`
	Pagination      PaginationInfo         `json:"pagination"`
}
