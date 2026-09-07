-- Adds provenance columns to signals_log.
--
-- `source` lets the daily auto-generation cap distinguish its own
-- signals from ones generated on demand or pushed in by n8n, and
-- `model` records which model produced a recommendation.
--
-- Both statements are idempotent; the backend also applies them at
-- startup so existing installs converge without running this by hand.

ALTER TABLE signals_log ADD COLUMN IF NOT EXISTS model TEXT;
ALTER TABLE signals_log ADD COLUMN IF NOT EXISTS source TEXT NOT NULL DEFAULT 'manual';
