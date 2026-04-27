-- Create cart table
CREATE TABLE IF NOT EXISTS carts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_carts_user_id ON carts(user_id);

-- Create cart_items table
CREATE TABLE IF NOT EXISTS cart_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cart_id UUID NOT NULL REFERENCES carts(id) ON DELETE CASCADE,
    course_id UUID NOT NULL,
    billing_type VARCHAR(20) NOT NULL DEFAULT 'ONE_TIME', -- ONE_TIME or MONTHLY
    price_cents BIGINT NOT NULL,
    currency VARCHAR(10) NOT NULL DEFAULT 'EGP',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(cart_id, course_id)
);

CREATE INDEX IF NOT EXISTS idx_cart_items_cart_id ON cart_items(cart_id);
CREATE INDEX IF NOT EXISTS idx_cart_items_course_id ON cart_items(course_id);

-- Create subscriptions table
CREATE TABLE IF NOT EXISTS subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    course_id UUID NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE', -- ACTIVE, CANCELLED, SUSPENDED, EXPIRED
    price_cents BIGINT NOT NULL,
    currency VARCHAR(10) NOT NULL DEFAULT 'EGP',
    billing_cycle VARCHAR(20) NOT NULL DEFAULT 'MONTHLY',
    next_billing_date TIMESTAMPTZ NOT NULL,
    last_billing_date TIMESTAMPTZ,
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    cancelled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(user_id, course_id)
);

CREATE INDEX IF NOT EXISTS idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_course_id ON subscriptions(course_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_status ON subscriptions(status);
CREATE INDEX IF NOT EXISTS idx_subscriptions_next_billing_date ON subscriptions(next_billing_date);

-- Update payment_orders to support multiple courses and subscriptions
ALTER TABLE payment_orders 
    DROP COLUMN IF EXISTS course_id,
    ADD COLUMN IF NOT EXISTS order_type VARCHAR(20) NOT NULL DEFAULT 'SINGLE_COURSE', -- SINGLE_COURSE, CART_CHECKOUT, SUBSCRIPTION_RENEWAL
    ADD COLUMN IF NOT EXISTS subscription_id UUID REFERENCES subscriptions(id);

-- Create payment_order_items table for multi-course orders
CREATE TABLE IF NOT EXISTS payment_order_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_order_id UUID NOT NULL REFERENCES payment_orders(id) ON DELETE CASCADE,
    course_id UUID NOT NULL,
    price_cents BIGINT NOT NULL,
    currency VARCHAR(10) NOT NULL DEFAULT 'EGP',
    billing_type VARCHAR(20) NOT NULL DEFAULT 'ONE_TIME',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_payment_order_items_order_id ON payment_order_items(payment_order_id);
CREATE INDEX IF NOT EXISTS idx_payment_order_items_course_id ON payment_order_items(course_id);
