-- Rollback for migrations/0003_smart_advisor.sql.
--
-- Run only after deploying an image that predates 0003; the API
-- re-applies every migration on boot, so a newer image would put all
-- of this straight back. model and source are NOT dropped: older images
-- read them.
--
-- Signals of kind 'review' and verdicts named CUT_LOSS are removed
-- first, because the older UI and API do not know them.

BEGIN;
DELETE FROM signals_log WHERE kind = 'review';
UPDATE signals_log SET signal_type = 'SELL' WHERE signal_type = 'CUT_LOSS';
ALTER TABLE signals_log
    DROP COLUMN IF EXISTS kind,
    DROP COLUMN IF EXISTS confidence,
    DROP COLUMN IF EXISTS horizon_days,
    DROP COLUMN IF EXISTS key_factors,
    DROP COLUMN IF EXISTS amount_bhd,
    DROP COLUMN IF EXISTS amount_grams,
    DROP COLUMN IF EXISTS levels,
    DROP COLUMN IF EXISTS news,
    DROP COLUMN IF EXISTS news_checked,
    DROP COLUMN IF EXISTS deviates,
    DROP COLUMN IF EXISTS suggested_action,
    DROP COLUMN IF EXISTS plan_text,
    DROP COLUMN IF EXISTS baseline;
DROP TABLE IF EXISTS ai_settings;
COMMIT;
