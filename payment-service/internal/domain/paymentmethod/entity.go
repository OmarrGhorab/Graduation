package paymentmethod

import (
	"time"

	"github.com/google/uuid"
)

type PaymentType string

const (
	PaymentTypeCard   PaymentType = "CARD"
	PaymentTypeWallet PaymentType = "WALLET"
)

type PaymentMethod struct {
	ID          uuid.UUID   `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID      uuid.UUID   `gorm:"type:uuid;not null;index"`
	PaymentType PaymentType `gorm:"type:varchar(20);not null;default:'CARD'"`
	Token       string      `gorm:"type:varchar(255);not null"` // Tokenized payment method
	LastFour    string      `gorm:"type:varchar(4)"`
	CardBrand   string      `gorm:"type:varchar(50)"`
	ExpiryMonth string      `gorm:"type:varchar(2)"`
	ExpiryYear  string      `gorm:"type:varchar(4)"`
	IsDefault   bool        `gorm:"not null;default:false"`
	IsActive    bool        `gorm:"not null;default:true"`
	CreatedAt   time.Time   `gorm:"not null;default:now()"`
	UpdatedAt   time.Time   `gorm:"not null;default:now()"`
}

func (PaymentMethod) TableName() string {
	return "payment_methods"
}
