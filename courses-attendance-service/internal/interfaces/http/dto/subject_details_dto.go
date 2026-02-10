package dto

import "github.com/google/uuid"

// SubjectDetailsResponse combines subject info with all courses in that subject
type SubjectDetailsResponse struct {
	Subject SubjectInfo   `json:"subject"`
	Courses []CourseCard  `json:"courses"`
}

// SubjectInfo contains basic subject information
type SubjectInfo struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Icon        string    `json:"icon"`
	TotalCourses int      `json:"totalCourses"`
}

// CourseCard contains course summary for subject view
type CourseCard struct {
	ID                uuid.UUID      `json:"id"`
	Title             string         `json:"title"`
	Description       string         `json:"description"`
	TeacherID         uuid.UUID      `json:"teacherId"`
	TeacherName       string         `json:"teacherName,omitempty"`
	TeacherProfileImg string         `json:"teacherProfileImg,omitempty"`
	DeliveryType      string         `json:"deliveryType"`
	LocationName      string         `json:"locationName,omitempty"`
	TotalLessons      int            `json:"totalLessons"`
	Price             float64        `json:"price"`
	Currency          string         `json:"currency"`
	IsPaid            bool           `json:"isPaid"`
	BillingType       string         `json:"billingType"`
	Status            string         `json:"status"`
	Progress          *ProgressInfo  `json:"progress,omitempty"` // Student's progress in this course
}
