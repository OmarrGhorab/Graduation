package dto

type AddToCartRequest struct {
	CourseID    string `json:"courseId" validate:"required,uuid"`
	BillingType string `json:"billingType" validate:"required,oneof=ONE_TIME MONTHLY"`
}

type RemoveFromCartRequest struct {
	CourseID string `json:"courseId" validate:"required,uuid"`
}

type CartItemResponse struct {
	ID          string `json:"id"`
	CourseID    string `json:"courseId"`
	BillingType string `json:"billingType"`
	PriceCents  int64  `json:"priceCents"`
	Currency    string `json:"currency"`
}

type CartResponse struct {
	ID         string             `json:"id"`
	Items      []CartItemResponse `json:"items"`
	TotalCents int64              `json:"totalCents"`
	Currency   string             `json:"currency"`
}

type CheckoutCartRequest struct {
	PaymentMethod string `json:"paymentMethod" validate:"required,oneof=CARD WALLET"`
	PhoneNumber   string `json:"phoneNumber" validate:"required"`
	FirstName     string `json:"firstName" validate:"required"`
	LastName      string `json:"lastName" validate:"required"`
	Email         string `json:"email" validate:"required,email"`
	Apartment     string `json:"apartment" validate:"omitempty"`
	Floor         string `json:"floor" validate:"omitempty"`
	Building      string `json:"building" validate:"omitempty"`
	Street        string `json:"street" validate:"omitempty"`
	City          string `json:"city" validate:"omitempty"`
	State         string `json:"state" validate:"omitempty"`
	Country       string `json:"country" validate:"omitempty"`
	SaveCard      bool   `json:"saveCard" validate:"omitempty"`
}
