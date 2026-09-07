-- Adds silver alongside gold.
--
-- Silver keeps its own price table rather than joining gold_prices:
-- the two metals are quoted independently, a day may carry one and not
-- the other, and gold's working path stays untouched.
--
-- Purity splits the same way. Gold is traded in karat, silver only in
-- millesimal fineness — sterling is 925, there is no karat for it — so
-- items carry whichever column their metal uses, with a CHECK making
-- sure the right one is filled. Anything that is neither gold nor
-- silver fails both branches, so the constraint bounds metal_type too.
--
-- Every statement is idempotent; the backend applies this file at
-- startup, so existing installs converge without running it by hand.

CREATE TABLE IF NOT EXISTS silver_prices (
    id SERIAL PRIMARY KEY,
    price_date DATE UNIQUE NOT NULL,
    price_per_gram_999 NUMERIC NOT NULL,
    -- A 925 piece holds 925 parts pure per thousand against 999's 999,
    -- the same shape as gold's price_per_gram_24k * 22 / 24.
    price_per_gram_925 NUMERIC GENERATED ALWAYS AS (price_per_gram_999 * 925 / 999) STORED,
    price_per_gram_900 NUMERIC GENERATED ALWAYS AS (price_per_gram_999 * 900 / 999) STORED,
    source TEXT,
    created_at TIMESTAMP DEFAULT now()
);

ALTER TABLE gold_items ADD COLUMN IF NOT EXISTS purity_fineness NUMERIC;
ALTER TABLE gold_items ALTER COLUMN purity_karat DROP NOT NULL;

UPDATE gold_items SET metal_type = 'gold' WHERE metal_type IS NULL;
ALTER TABLE gold_items ALTER COLUMN metal_type SET NOT NULL;

ALTER TABLE gold_items DROP CONSTRAINT IF EXISTS chk_item_purity;
ALTER TABLE gold_items ADD CONSTRAINT chk_item_purity CHECK (
    (metal_type = 'gold'   AND purity_karat    IS NOT NULL) OR
    (metal_type = 'silver' AND purity_fineness IS NOT NULL)
);

ALTER TABLE signals_log ADD COLUMN IF NOT EXISTS metal TEXT NOT NULL DEFAULT 'gold';

-- The view is dropped rather than replaced: CREATE OR REPLACE VIEW can
-- only append columns, and metal_type and purity_fineness belong next
-- to the other item facts.
DROP VIEW IF EXISTS v_portfolio_summary;

CREATE VIEW v_portfolio_summary AS
SELECT
    gi.id,
    gi.item_name,
    gi.purchase_date,
    gi.metal_type,
    gi.purity_karat,
    gi.purity_fineness,
    gi.weight_grams,
    gi.price_paid_total,
    gi.price_per_gram_paid,
    CASE gi.metal_type WHEN 'silver' THEN ls.price_date ELSE lg.price_date END AS latest_price_date,
    ppg.price_per_gram AS current_price_per_gram,
    (gi.weight_grams * ppg.price_per_gram) AS current_value,
    ((gi.weight_grams * ppg.price_per_gram) - gi.price_paid_total) AS gain_loss,
    ROUND(
        CAST(
            (((gi.weight_grams * ppg.price_per_gram) - gi.price_paid_total)
             / NULLIF(gi.price_paid_total, 0)) * 100
        AS NUMERIC),
    2) AS gain_loss_pct
FROM gold_items gi
LEFT JOIN LATERAL (
    SELECT * FROM gold_prices ORDER BY price_date DESC LIMIT 1
) lg ON gi.metal_type = 'gold'
LEFT JOIN LATERAL (
    SELECT * FROM silver_prices ORDER BY price_date DESC LIMIT 1
) ls ON gi.metal_type = 'silver'
-- The per-gram price is computed once here instead of being pasted
-- into current_value, gain_loss, and gain_loss_pct in turn, which is
-- how the previous version came to hold four copies of one CASE.
CROSS JOIN LATERAL (
    SELECT CASE gi.metal_type
        WHEN 'silver' THEN CASE gi.purity_fineness
            WHEN 999 THEN ls.price_per_gram_999
            WHEN 925 THEN ls.price_per_gram_925
            WHEN 900 THEN ls.price_per_gram_900
            ELSE (ls.price_per_gram_999 * gi.purity_fineness / 999)
        END
        ELSE CASE gi.purity_karat
            WHEN 24 THEN lg.price_per_gram_24k
            WHEN 22 THEN lg.price_per_gram_22k
            WHEN 21 THEN lg.price_per_gram_21k
            WHEN 18 THEN lg.price_per_gram_18k
            ELSE (lg.price_per_gram_24k * gi.purity_karat / 24)
        END
    END AS price_per_gram
) ppg;
