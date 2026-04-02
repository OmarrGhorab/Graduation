package payment

import (
	"time"

	"github.com/google/uuid"
)

type OrderStatus string

const (
	OrderStatusPending  OrderStatus = "PENDING"
	OrderStatusPaid     OrderStatus = "PAID"
	OrderStatusFailed   OrderStatus = "FAILED"
	OrderStatusRefunded OrderStatus = "REFUNDED"
)

type OrderType string

const (
	OrderTypeSingleCourse       OrderType = "SINGLE_COURSE"
	OrderTypeCartCheckout       OrderType = "CART_CHECKOUT"
	OrderTypeSubscriptionRenewal OrderType = "SUBSCRIPTION_RENEWAL"
)

type PaymentOrder struct {
	ID              uuid.UUID          `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID          uuid.UUID          `gorm:"type:uuid;not null;index"`
	AmountCents     int64              `gorm:"not null"`
	Currency        string             `gorm:"type:varchar(10);not null;default:'EGP'"`
	Status          OrderStatus        `gorm:"type:varchar(20);not null;default:'PENDING'"`
	OrderType       OrderType          `gorm:"type:varchar(20);not null;default:'SINGLE_COURSE'"`
	SubscriptionID  *uuid.UUID         `gorm:"type:uuid"`
	PaymentMethodID *uuid.UUID         `gorm:"type:uuid"`
	IsAutoCharge    bool               `gorm:"not null;default:false"`
	PaymobOrderID   string             `gorm:"type:varchar(100);index"`
	PaymentMethod   string             `gorm:"type:varchar(50)"`
	CreatedAt       time.Time          `gorm:"not null;default:now()"`
	UpdatedAt       time.Time          `gorm:"not null;default:now()"`
	Items           []PaymentOrderItem `gorm:"foreignKey:PaymentOrderID"`
}

type PaymentOrderItem struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	PaymentOrderID uuid.UUID `gorm:"type:uuid;not null"`
	CourseID       uuid.UUID `gorm:"type:uuid;not null"`
	PriceCents     int64     `gorm:"not null"`
	Currency       string    `gorm:"type:varchar(10);not null;default:'EGP'"`
	BillingType    string    `gorm:"type:varchar(20);not null;default:'ONE_TIME'"`
	CreatedAt      time.Time `gorm:"not null;default:now()"`
}

func (PaymentOrderItem) TableName() string {
	return "payment_order_items"
}

func (PaymentOrder) TableName() string {
	return "payment_orders"
}

type PaymentTransaction struct {
	ID                  uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	PaymentOrderID      uuid.UUID `gorm:"type:uuid;not null;index"`
	PaymobTransactionID string    `gorm:"type:varchar(100);not null;index"`
	PaymentMethod       string    `gorm:"type:varchar(50)"`
	AmountCents         int64     `gorm:"not null"`
	Success             bool      `gorm:"not null"`
	RawResponse         []byte    `gorm:"type:jsonb"`
	CreatedAt           time.Time `gorm:"not null;default:now()"`
}

func (PaymentTransaction) TableName() string {
	return "payment_transactions"
}
