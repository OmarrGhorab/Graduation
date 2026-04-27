package cart

import (
	"time"

	"github.com/google/uuid"
)

type BillingType string

const (
	BillingTypeOneTime BillingType = "ONE_TIME"
	BillingTypeMonthly BillingType = "MONTHLY"
)

type Cart struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;unique"`
	CreatedAt time.Time `gorm:"not null;default:now()"`
	UpdatedAt time.Time `gorm:"not null;default:now()"`
	Items     []CartItem `gorm:"foreignKey:CartID"`
}

func (Cart) TableName() string {
	return "carts"
}

type CartItem struct {
	ID          uuid.UUID   `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	CartID      uuid.UUID   `gorm:"type:uuid;not null"`
	CourseID    uuid.UUID   `gorm:"type:uuid;not null"`
	BillingType BillingType `gorm:"type:varchar(20);not null;default:'ONE_TIME'"`
	PriceCents  int64       `gorm:"not null"`
	Currency    string      `gorm:"type:varchar(10);not null;default:'EGP'"`
	CreatedAt   time.Time   `gorm:"not null;default:now()"`
}

func (CartItem) TableName() string {
	return "cart_items"
}
