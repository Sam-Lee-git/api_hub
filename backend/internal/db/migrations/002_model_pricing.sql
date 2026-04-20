-- +migrate Up

CREATE TABLE providers (
    id         SERIAL PRIMARY KEY,
    name       VARCHAR(50)  NOT NULL UNIQUE,
    base_url   TEXT         NOT NULL,
    api_key    TEXT         NOT NULL DEFAULT '',
    status     VARCHAR(20)  NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE models (
    id                     SERIAL PRIMARY KEY,
    provider_id            INT          NOT NULL REFERENCES providers(id),
    model_id               VARCHAR(100) NOT NULL UNIQUE,
    display_name           VARCHAR(100) NOT NULL,
    input_credits_per_1k   BIGINT       NOT NULL DEFAULT 0,
    output_credits_per_1k  BIGINT       NOT NULL DEFAULT 0,
    context_window         INT          NOT NULL DEFAULT 128000,
    supports_streaming     BOOLEAN      NOT NULL DEFAULT TRUE,
    supports_vision        BOOLEAN      NOT NULL DEFAULT FALSE,
    status                 VARCHAR(20)  NOT NULL DEFAULT 'active',
    created_at             TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_models_model_id ON models(model_id) WHERE status = 'active';

CREATE TABLE usage_records (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT      NOT NULL REFERENCES users(id),
    api_key_id      BIGINT      NOT NULL REFERENCES api_keys(id),
    model_id        INT         NOT NULL REFERENCES models(id),
    request_id      VARCHAR(50) NOT NULL UNIQUE,
    input_tokens    INT         NOT NULL DEFAULT 0,
    output_tokens   INT         NOT NULL DEFAULT 0,
    total_tokens    INT         NOT NULL DEFAULT 0,
    credits_charged BIGINT      NOT NULL DEFAULT 0,
    status          VARCHAR(20) NOT NULL DEFAULT 'success',
    latency_ms      INT         NOT NULL DEFAULT 0,
    error_message   TEXT        NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_usage_user_time  ON usage_records(user_id, created_at DESC);
CREATE INDEX idx_usage_model_time ON usage_records(model_id, created_at DESC);
CREATE INDEX idx_usage_created_at ON usage_records(created_at DESC);

-- Seed providers
INSERT INTO providers (name, base_url) VALUES
    ('openai',    'https://api.openai.com/v1'),
    ('anthropic', 'https://api.anthropic.com'),
    ('google',    'https://generativelanguage.googleapis.com/v1beta'),
    ('alibaba',   'https://dashscope.aliyuncs.com/compatible-mode/v1');

-- Seed models (1 credit = 0.001 CNY; exchange rate ~7.3 CNY/USD)
-- GPT-4o: $5/1M input, $15/1M output → 36.5/1K input, 109.5/1K output credits
INSERT INTO models (provider_id, model_id, display_name, input_credits_per_1k, output_credits_per_1k, context_window, supports_vision) VALUES
    (1, 'gpt-4o',                'GPT-4o',              37, 111, 128000, true),
    (1, 'gpt-4o-mini',           'GPT-4o Mini',          1,   4, 128000, true),
    (1, 'gpt-4-turbo',           'GPT-4 Turbo',         73, 219, 128000, true),
    (1, 'gpt-3.5-turbo',         'GPT-3.5 Turbo',        1,   2,  16385, false),
    (2, 'claude-3-5-sonnet-20241022', 'Claude 3.5 Sonnet', 22, 110, 200000, true),
    (2, 'claude-3-5-haiku-20241022',  'Claude 3.5 Haiku',   1,   4, 200000, true),
    (2, 'claude-3-opus-20240229',     'Claude 3 Opus',     110, 549, 200000, true),
    (3, 'gemini-1.5-pro',        'Gemini 1.5 Pro',       9,  27, 1000000, true),
    (3, 'gemini-1.5-flash',      'Gemini 1.5 Flash',     1,   3, 1000000, true),
    (3, 'gemini-2.0-flash',      'Gemini 2.0 Flash',     1,   3, 1000000, true),
    (4, 'qwen-max',              'Qwen Max',             36, 108, 32000, false),
    (4, 'qwen-plus',             'Qwen Plus',             4,  12, 131072, false),
    (4, 'qwen-turbo',            'Qwen Turbo',            1,   3, 131072, false);
