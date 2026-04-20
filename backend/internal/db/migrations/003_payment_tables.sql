-- +migrate Up

CREATE TABLE payment_orders (
    id                BIGSERIAL PRIMARY KEY,
    user_id           BIGINT       NOT NULL REFERENCES users(id),
    order_no          VARCHAR(64)  NOT NULL UNIQUE,
    channel           VARCHAR(20)  NOT NULL,
    amount_cny        BIGINT       NOT NULL,
    credits_to_add    BIGINT       NOT NULL,
    status            VARCHAR(20)  NOT NULL DEFAULT 'pending',
    provider_order_no VARCHAR(100),
    paid_at           TIMESTAMPTZ,
    expires_at        TIMESTAMPTZ  NOT NULL,
    metadata          JSONB,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_orders_user     ON payment_orders(user_id, created_at DESC);
CREATE INDEX idx_orders_order_no ON payment_orders(order_no);
CREATE INDEX idx_orders_pending  ON payment_orders(status) WHERE status = 'pending';

CREATE TABLE credit_packages (
    id            SERIAL PRIMARY KEY,
    name          VARCHAR(100) NOT NULL,
    amount_cny    BIGINT       NOT NULL,
    credits       BIGINT       NOT NULL,
    bonus_credits BIGINT       NOT NULL DEFAULT 0,
    is_active     BOOLEAN      NOT NULL DEFAULT TRUE,
    display_order INT          NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Seed packages: 1 CNY = 100 fen; 10 CNY = 10000 credits
INSERT INTO credit_packages (name, amount_cny, credits, bonus_credits, display_order) VALUES
    ('入门包 10元',   1000, 10000,    0, 1),
    ('标准包 50元',   5000, 50000, 5000, 2),
    ('专业包 100元', 10000, 100000, 20000, 3);
