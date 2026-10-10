-- The advisor: sized, news-aware verdicts and reviews of typed plans.
--
-- Everything here is additive and idempotent, because the API applies
-- every migration on every boot (see applyMigrations).

-- model and source were added by migrations/0001_add_signal_source.sql,
-- but that file sits in a subdirectory `go:embed *.sql` never reads, so
-- a fresh install never got them. Repeating them here is a no-op on
-- installs that already have them.
ALTER TABLE signals_log ADD COLUMN IF NOT EXISTS model TEXT;
ALTER TABLE signals_log ADD COLUMN IF NOT EXISTS source TEXT NOT NULL DEFAULT 'manual';

ALTER TABLE signals_log ADD COLUMN IF NOT EXISTS kind TEXT NOT NULL DEFAULT 'analysis';
ALTER TABLE signals_log ADD COLUMN IF NOT EXISTS confidence NUMERIC;
ALTER TABLE signals_log ADD COLUMN IF NOT EXISTS horizon_days INTEGER;
ALTER TABLE signals_log ADD COLUMN IF NOT EXISTS key_factors JSONB;
ALTER TABLE signals_log ADD COLUMN IF NOT EXISTS amount_bhd NUMERIC;
ALTER TABLE signals_log ADD COLUMN IF NOT EXISTS amount_grams NUMERIC;
ALTER TABLE signals_log ADD COLUMN IF NOT EXISTS levels JSONB;
ALTER TABLE signals_log ADD COLUMN IF NOT EXISTS news JSONB;
ALTER TABLE signals_log ADD COLUMN IF NOT EXISTS news_checked BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE signals_log ADD COLUMN IF NOT EXISTS deviates BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE signals_log ADD COLUMN IF NOT EXISTS suggested_action TEXT;
ALTER TABLE signals_log ADD COLUMN IF NOT EXISTS plan_text TEXT;
ALTER TABLE signals_log ADD COLUMN IF NOT EXISTS baseline JSONB;

-- One row, id = 1. The CHECK keeps it that way.
CREATE TABLE IF NOT EXISTS ai_settings (
    id INTEGER PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    monthly_budget_bhd NUMERIC NOT NULL DEFAULT 0,
    reserve_bhd NUMERIC NOT NULL DEFAULT 0,
    target_gold_pct NUMERIC NOT NULL DEFAULT 80,
    spread_pct_gold NUMERIC NOT NULL DEFAULT 1,
    spread_pct_silver NUMERIC NOT NULL DEFAULT 1,
    min_fee_bhd NUMERIC NOT NULL DEFAULT 0,
    stop_loss_pct NUMERIC NOT NULL DEFAULT 15,
    news_enabled BOOLEAN NOT NULL DEFAULT true,
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);
INSERT INTO ai_settings (id) VALUES (1) ON CONFLICT (id) DO NOTHING;
