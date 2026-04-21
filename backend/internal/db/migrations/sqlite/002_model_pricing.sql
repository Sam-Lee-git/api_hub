CREATE TABLE IF NOT EXISTS providers (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT NOT NULL UNIQUE,
    base_url   TEXT NOT NULL,
    api_key    TEXT NOT NULL DEFAULT '',
    status     TEXT NOT NULL DEFAULT 'active',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS models (
    id                    INTEGER PRIMARY KEY AUTOINCREMENT,
    provider_id           INTEGER NOT NULL REFERENCES providers(id),
    model_id              TEXT    NOT NULL UNIQUE,
    display_name          TEXT    NOT NULL,
    input_credits_per_1k  INTEGER NOT NULL DEFAULT 0,
    output_credits_per_1k INTEGER NOT NULL DEFAULT 0,
    context_window        INTEGER NOT NULL DEFAULT 128000,
    supports_streaming    INTEGER NOT NULL DEFAULT 1,
    supports_vision       INTEGER NOT NULL DEFAULT 0,
    status                TEXT    NOT NULL DEFAULT 'active',
    created_at            TEXT    NOT NULL DEFAULT (datetime('now')),
    updated_at            TEXT    NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_models_status ON models(model_id, status);

CREATE TABLE IF NOT EXISTS usage_records (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id         INTEGER NOT NULL REFERENCES users(id),
    api_key_id      INTEGER NOT NULL REFERENCES api_keys(id),
    model_id        INTEGER NOT NULL REFERENCES models(id),
    request_id      TEXT    NOT NULL UNIQUE,
    input_tokens    INTEGER NOT NULL DEFAULT 0,
    output_tokens   INTEGER NOT NULL DEFAULT 0,
    total_tokens    INTEGER NOT NULL DEFAULT 0,
    credits_charged INTEGER NOT NULL DEFAULT 0,
    status          TEXT    NOT NULL DEFAULT 'success',
    latency_ms      INTEGER NOT NULL DEFAULT 0,
    error_message   TEXT    NOT NULL DEFAULT '',
    created_at      TEXT    NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_usage_user ON usage_records(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_usage_model ON usage_records(model_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_usage_time ON usage_records(created_at DESC);

INSERT OR IGNORE INTO providers (name, base_url) VALUES
    ('openai',    'https://api.openai.com/v1'),
    ('anthropic', 'https://api.anthropic.com'),
    ('google',    'https://generativelanguage.googleapis.com/v1beta'),
    ('alibaba',   'https://dashscope.aliyuncs.com/compatible-mode/v1');

INSERT OR IGNORE INTO models (provider_id, model_id, display_name, input_credits_per_1k, output_credits_per_1k, context_window, supports_vision) VALUES
    (1, 'gpt-4o',                     'GPT-4o',              37, 111, 128000, 1),
    (1, 'gpt-4o-mini',                'GPT-4o Mini',          1,   4, 128000, 1),
    (1, 'gpt-4-turbo',                'GPT-4 Turbo',         73, 219, 128000, 1),
    (1, 'gpt-3.5-turbo',              'GPT-3.5 Turbo',        1,   2,  16385, 0),
    (2, 'claude-3-5-sonnet-20241022', 'Claude 3.5 Sonnet',   22, 110, 200000, 1),
    (2, 'claude-3-5-haiku-20241022',  'Claude 3.5 Haiku',     1,   4, 200000, 1),
    (2, 'claude-3-opus-20240229',     'Claude 3 Opus',       110, 549, 200000, 1),
    (3, 'gemini-1.5-pro',             'Gemini 1.5 Pro',        9,  27, 1000000, 1),
    (3, 'gemini-1.5-flash',           'Gemini 1.5 Flash',      1,   3, 1000000, 1),
    (3, 'gemini-2.0-flash',           'Gemini 2.0 Flash',      1,   3, 1000000, 1),
    (4, 'qwen-max',                   'Qwen Max',             36, 108,  32000, 0),
    (4, 'qwen-plus',                  'Qwen Plus',             4,  12, 131072, 0),
    (4, 'qwen-turbo',                 'Qwen Turbo',            1,   3, 131072, 0);
