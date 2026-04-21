CREATE TABLE IF NOT EXISTS payment_orders (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id          INTEGER NOT NULL REFERENCES users(id),
    order_no         TEXT    NOT NULL UNIQUE,
    channel          TEXT    NOT NULL,
    amount_cny       INTEGER NOT NULL,
    credits_to_add   INTEGER NOT NULL,
    status           TEXT    NOT NULL DEFAULT 'pending',
    provider_order_no TEXT   NOT NULL DEFAULT '',
    paid_at          TEXT,
    expires_at       TEXT    NOT NULL,
    created_at       TEXT    NOT NULL DEFAULT (datetime('now')),
    updated_at       TEXT    NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_orders_user ON payment_orders(user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS credit_packages (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    name          TEXT    NOT NULL,
    amount_cny    INTEGER NOT NULL,
    credits       INTEGER NOT NULL,
    bonus_credits INTEGER NOT NULL DEFAULT 0,
    is_active     INTEGER NOT NULL DEFAULT 1,
    display_order INTEGER NOT NULL DEFAULT 0,
    created_at    TEXT    NOT NULL DEFAULT (datetime('now'))
);

INSERT OR IGNORE INTO credit_packages (id, name, amount_cny, credits, bonus_credits, display_order) VALUES
    (1, '入门套餐 10元',  1000,  10000,     0, 1),
    (2, '标准套餐 50元',  5000,  50000,  5000, 2),
    (3, '高级套餐 100元', 10000, 100000, 20000, 3);
