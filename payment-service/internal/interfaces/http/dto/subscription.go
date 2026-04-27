package dto

type SubscriptionResponse struct {
	ID              string  `json:"id"`
	UserID          string  `json:"userId"`
	CourseID        string  `json:"courseId"`
	Status          string  `json:"status"`
	PriceCents      int64   `json:"priceCents"`
	Currency        string  `json:"currency"`
	BillingCycle    string  `json:"billingCycle"`
	NextBillingDate string  `json:"nextBillingDate"`
	LastBillingDate *string `json:"lastBillingDate,omitempty"`
	StartedAt       string  `json:"startedAt"`
	CancelledAt     *string `json:"cancelledAt,omitempty"`
}

type CancelSubscriptionRequest struct {
	SubscriptionID string `json:"subscriptionId" validate:"required,uuid"`
}
