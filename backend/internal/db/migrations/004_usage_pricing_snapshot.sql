-- +migrate Up

ALTER TABLE usage_records
    ADD COLUMN IF NOT EXISTS input_credits_per_1k_snapshot BIGINT NOT NULL DEFAULT 0;

ALTER TABLE usage_records
    ADD COLUMN IF NOT EXISTS output_credits_per_1k_snapshot BIGINT NOT NULL DEFAULT 0;
