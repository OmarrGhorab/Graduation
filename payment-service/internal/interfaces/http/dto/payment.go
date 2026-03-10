package dto

type CreatePaymentRequest struct {
	CourseID      string `json:"courseId" validate:"required,uuid"`
	PaymentMethod string `json:"paymentMethod" validate:"required,oneof=CARD WALLET"`
	PhoneNumber   string `json:"phoneNumber" validate:"required_if=PaymentMethod WALLET"`
	FirstName     string `json:"firstName" validate:"required"`
	LastName      string `json:"lastName" validate:"required"`
	Email         string `json:"email" validate:"required,email"`
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
