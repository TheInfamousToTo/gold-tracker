-- Rollback for 0002_add_silver.sql.
--
-- Run this ONLY if you roll the API image back to a pre-silver build.
--
-- Rolling the image back on its own is not enough. 0002 widened
-- v_portfolio_summary from 12 columns to 14, and the old code reads it
-- with SELECT * and a positional scan of 12 — so the old image serves
-- 500s on /api/portfolio against the new view. This restores the view
-- the old code expects.
--
-- It deliberately does NOT drop silver_prices, purity_fineness, or
-- signals_log.metal. Those are additive, invisible to the old code, and
-- dropping them would throw away real data. Silver items already
-- entered will disappear from the portfolio while rolled back, because
-- the old view cannot price them — they return when you roll forward.

DROP VIEW IF EXISTS v_portfolio_summary;

CREATE OR REPLACE VIEW v_portfolio_summary AS
SELECT
    gi.id,
    gi.item_name,
    gi.purchase_date,
    gi.purity_karat,
    gi.weight_grams,
    gi.price_paid_total,
    gi.price_per_gram_paid,
    lp.price_date AS latest_price_date,
    CASE gi.purity_karat
        WHEN 24 THEN lp.price_per_gram_24k
        WHEN 22 THEN lp.price_per_gram_22k
        WHEN 21 THEN lp.price_per_gram_21k
        WHEN 18 THEN lp.price_per_gram_18k
        ELSE (lp.price_per_gram_24k * gi.purity_karat / 24)
    END AS current_price_per_gram,
    (gi.weight_grams * CASE gi.purity_karat
        WHEN 24 THEN lp.price_per_gram_24k
        WHEN 22 THEN lp.price_per_gram_22k
        WHEN 21 THEN lp.price_per_gram_21k
        WHEN 18 THEN lp.price_per_gram_18k
        ELSE (lp.price_per_gram_24k * gi.purity_karat / 24)
    END) AS current_value,
    ((gi.weight_grams * CASE gi.purity_karat
        WHEN 24 THEN lp.price_per_gram_24k
        WHEN 22 THEN lp.price_per_gram_22k
        WHEN 21 THEN lp.price_per_gram_21k
        WHEN 18 THEN lp.price_per_gram_18k
        ELSE (lp.price_per_gram_24k * gi.purity_karat / 24)
    END) - gi.price_paid_total) AS gain_loss,
    ROUND(
        CAST(
            (((gi.weight_grams * CASE gi.purity_karat
                WHEN 24 THEN lp.price_per_gram_24k
                WHEN 22 THEN lp.price_per_gram_22k
                WHEN 21 THEN lp.price_per_gram_21k
                WHEN 18 THEN lp.price_per_gram_18k
                ELSE (lp.price_per_gram_24k * gi.purity_karat / 24)
            END) - gi.price_paid_total) / gi.price_paid_total) * 100
        AS NUMERIC),
    2) AS gain_loss_pct
FROM gold_items gi
LEFT JOIN LATERAL (SELECT * FROM gold_prices ORDER BY price_date DESC LIMIT 1) lp ON true;
