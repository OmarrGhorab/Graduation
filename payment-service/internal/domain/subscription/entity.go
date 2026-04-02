package subscription

import (
	"time"

	"github.com/google/uuid"
)

type SubscriptionStatus string

const (
	StatusActive    SubscriptionStatus = "ACTIVE"
	StatusCancelled SubscriptionStatus = "CANCELLED"
	StatusSuspended SubscriptionStatus = "SUSPENDED"
	StatusExpired   SubscriptionStatus = "EXPIRED"
)

type BillingCycle string

const (
	BillingCycleMonthly BillingCycle = "MONTHLY"
)

type Subscription struct {
	ID              uuid.UUID          `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID          uuid.UUID          `gorm:"type:uuid;not null"`
	CourseID        uuid.UUID          `gorm:"type:uuid;not null"`
	Status          SubscriptionStatus `gorm:"type:varchar(20);not null;default:'ACTIVE'"`
	PriceCents      int64              `gorm:"not null"`
	Currency        string             `gorm:"type:varchar(10);not null;default:'EGP'"`
	BillingCycle    BillingCycle       `gorm:"type:varchar(20);not null;default:'MONTHLY'"`
	PaymentMethodID *uuid.UUID         `gorm:"type:uuid"`
	NextBillingDate time.Time          `gorm:"not null"`
	LastBillingDate *time.Time         `gorm:""`
	StartedAt       time.Time          `gorm:"not null;default:now()"`
	CancelledAt     *time.Time         `gorm:""`
	CreatedAt       time.Time          `gorm:"not null;default:now()"`
	UpdatedAt       time.Time          `gorm:"not null;default:now()"`
}

func (Subscription) TableName() string {
	return "subscriptions"
}
