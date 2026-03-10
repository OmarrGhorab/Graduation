package postgres

import (
	"context"

	"github.com/OmarrGhorab/payment-service/internal/domain/payment"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PaymentRepository struct {
	db *Database
}

func NewPaymentRepository(db *Database) *PaymentRepository {
	return &PaymentRepository{db: db}
}

func (r *PaymentRepository) CreateOrder(ctx context.Context, order *payment.PaymentOrder) error {
	return r.db.WithContext(ctx).Create(order).Error
}

func (r *PaymentRepository) GetOrderByID(ctx context.Context, id uuid.UUID) (*payment.PaymentOrder, error) {
	var order payment.PaymentOrder
	err := r.db.WithContext(ctx).First(&order, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *PaymentRepository) GetOrderByPaymobID(ctx context.Context, paymobOrderID string) (*payment.PaymentOrder, error) {
	var order payment.PaymentOrder
	err := r.db.WithContext(ctx).First(&order, "paymob_order_id = ?", paymobOrderID).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *PaymentRepository) UpdateOrderStatus(ctx context.Context, orderID uuid.UUID, status payment.OrderStatus, paymobOrderID *string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if paymobOrderID != nil {
		updates["paymob_order_id"] = *paymobOrderID
	}
	return r.db.WithContext(ctx).Model(&payment.PaymentOrder{}).Where("id = ?", orderID).Updates(updates).Error
}

func (r *PaymentRepository) CreateTransaction(ctx context.Context, tx *payment.PaymentTransaction) error {
	return r.db.WithContext(ctx).Create(tx).Error
}

func (r *PaymentRepository) GetTransactionByPaymobID(ctx context.Context, paymobTransactionID string) (*payment.PaymentTransaction, error) {
	var tx payment.PaymentTransaction
	err := r.db.WithContext(ctx).First(&tx, "paymob_transaction_id = ?", paymobTransactionID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &tx, nil
}
