-- Create payment_methods table for storing tokenized payment information
CREATE TABLE IF NOT EXISTS payment_methods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    payment_type VARCHAR(20) NOT NULL DEFAULT 'CARD', -- CARD, WALLET
    token VARCHAR(255) NOT NULL, -- Tokenized payment method from Paymob
    last_four VARCHAR(4), -- Last 4 digits of card
    card_brand VARCHAR(50), -- Visa, Mastercard, etc.
    expiry_month VARCHAR(2),
    expiry_year VARCHAR(4),
    is_default BOOLEAN NOT NULL DEFAULT false,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_payment_methods_user_id ON payment_methods(user_id);
CREATE INDEX IF NOT EXISTS idx_payment_methods_is_default ON payment_methods(user_id, is_default) WHERE is_default = true;

-- Add payment_method_id to subscriptions for automatic billing
ALTER TABLE subscriptions 
    ADD COLUMN IF NOT EXISTS payment_method_id UUID REFERENCES payment_methods(id);

CREATE INDEX IF NOT EXISTS idx_subscriptions_payment_method ON subscriptions(payment_method_id);

-- Add metadata to payment_orders for tracking payment method used
ALTER TABLE payment_orders
    ADD COLUMN IF NOT EXISTS payment_method_id UUID REFERENCES payment_methods(id),
    ADD COLUMN IF NOT EXISTS is_auto_charge BOOLEAN NOT NULL DEFAULT false;
