ALTER TABLE usage_records ADD COLUMN input_credits_per_1k_snapshot INTEGER NOT NULL DEFAULT 0;
ALTER TABLE usage_records ADD COLUMN output_credits_per_1k_snapshot INTEGER NOT NULL DEFAULT 0;
