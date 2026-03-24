package dto

type CreatePaymentRequest struct {
	CourseID      string `json:"courseId" validate:"required,uuid"`
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
}

type CreatePaymentResponse struct {
	PaymentURL     string `json:"paymentUrl"`
	PaymentOrderID string `json:"paymentOrderId"`
}

type PaymentStatusResponse struct {
	OrderID   string `json:"orderId"`
	UserID    string `json:"userId"`
	CourseID  string `json:"courseId"`
	Amount    int64  `json:"amount"`
	Currency  string `json:"currency"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
}
