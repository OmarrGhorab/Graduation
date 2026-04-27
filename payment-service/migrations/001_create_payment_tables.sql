-- Create payment_orders table
CREATE TABLE IF NOT EXISTS payment_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    course_id UUID NOT NULL,
    amount_cents BIGINT NOT NULL,
    currency VARCHAR(10) NOT NULL DEFAULT 'EGP',
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    paymob_order_id VARCHAR(100),
    payment_method VARCHAR(50),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_payment_orders_user_id ON payment_orders(user_id);
CREATE INDEX IF NOT EXISTS idx_payment_orders_course_id ON payment_orders(course_id);
CREATE INDEX IF NOT EXISTS idx_payment_orders_paymob_order_id ON payment_orders(paymob_order_id);

-- Create payment_transactions table
CREATE TABLE IF NOT EXISTS payment_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_order_id UUID NOT NULL REFERENCES payment_orders(id),
    paymob_transaction_id VARCHAR(100) NOT NULL,
    payment_method VARCHAR(50),
    amount_cents BIGINT NOT NULL,
    success BOOLEAN NOT NULL,
    raw_response JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_payment_transactions_order_id ON payment_transactions(payment_order_id);
CREATE INDEX IF NOT EXISTS idx_payment_transactions_paymob_tx_id ON payment_transactions(paymob_transaction_id);
