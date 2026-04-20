-- +migrate Up

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE users (
    id            BIGSERIAL PRIMARY KEY,
    email         VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    display_name  VARCHAR(100) NOT NULL DEFAULT '',
    role          VARCHAR(20)  NOT NULL DEFAULT 'user',
    status        VARCHAR(20)  NOT NULL DEFAULT 'active',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ
);

CREATE INDEX idx_users_email ON users(email) WHERE deleted_at IS NULL;

CREATE TABLE credit_accounts (
    id           BIGSERIAL PRIMARY KEY,
    user_id      BIGINT       NOT NULL UNIQUE REFERENCES users(id),
    balance      BIGINT       NOT NULL DEFAULT 0,
    total_spent  BIGINT       NOT NULL DEFAULT 0,
    total_topped BIGINT       NOT NULL DEFAULT 0,
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE credit_transactions (
    id            BIGSERIAL PRIMARY KEY,
    user_id       BIGINT      NOT NULL REFERENCES users(id),
    type          VARCHAR(20) NOT NULL,
    amount        BIGINT      NOT NULL,
    balance_after BIGINT      NOT NULL,
    ref_id        VARCHAR(100),
    description   TEXT        NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_credit_tx_user ON credit_transactions(user_id, created_at DESC);

CREATE TABLE api_keys (
    id           BIGSERIAL PRIMARY KEY,
    user_id      BIGINT      NOT NULL REFERENCES users(id),
    key_hash     VARCHAR(64) NOT NULL UNIQUE,
    key_prefix   VARCHAR(12) NOT NULL,
    name         VARCHAR(100) NOT NULL DEFAULT '',
    status       VARCHAR(20) NOT NULL DEFAULT 'active',
    last_used_at TIMESTAMPTZ,
    expires_at   TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at   TIMESTAMPTZ
);

CREATE INDEX idx_api_keys_hash ON api_keys(key_hash) WHERE deleted_at IS NULL;
CREATE INDEX idx_api_keys_user ON api_keys(user_id)  WHERE deleted_at IS NULL;

CREATE TABLE refresh_tokens (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT      NOT NULL REFERENCES users(id),
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked    BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_refresh_tokens_hash ON refresh_tokens(token_hash) WHERE NOT revoked;
