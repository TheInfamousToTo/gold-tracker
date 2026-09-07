-- The original schema.
--
-- This was previously inlined in setup_gold_db.sh, which meant a fresh
-- install and a migrated one were built from two different sources.
-- Both now apply this directory in filename order, so there is one
-- description of the schema and no way for the two paths to diverge.
--
-- v_portfolio_summary is deliberately absent here: 0002 creates it in
-- its metal-aware form, and defining a gold-only version first would
-- only be something to drop moments later.

CREATE TABLE IF NOT EXISTS gold_items (
    id SERIAL PRIMARY KEY,
    purchase_date DATE NOT NULL,
    item_name TEXT NOT NULL,
    metal_type TEXT DEFAULT 'gold',
    purity_karat NUMERIC NOT NULL,
    weight_grams NUMERIC NOT NULL,
    price_paid_total NUMERIC NOT NULL,
    price_per_gram_paid NUMERIC GENERATED ALWAYS AS (price_paid_total / weight_grams) STORED,
    vendor TEXT,
    notes TEXT,
    created_at TIMESTAMP DEFAULT now()
);

CREATE TABLE IF NOT EXISTS gold_prices (
    id SERIAL PRIMARY KEY,
    price_date DATE UNIQUE NOT NULL,
    price_per_gram_24k NUMERIC NOT NULL,
    price_per_gram_22k NUMERIC GENERATED ALWAYS AS (price_per_gram_24k * 22 / 24) STORED,
    price_per_gram_21k NUMERIC GENERATED ALWAYS AS (price_per_gram_24k * 21 / 24) STORED,
    price_per_gram_18k NUMERIC GENERATED ALWAYS AS (price_per_gram_24k * 18 / 24) STORED,
    source TEXT,
    created_at TIMESTAMP DEFAULT now()
);

CREATE TABLE IF NOT EXISTS signals_log (
    id SERIAL PRIMARY KEY,
    signal_date TIMESTAMP DEFAULT now(),
    signal_type TEXT NOT NULL,
    reasoning TEXT,
    price_at_signal NUMERIC,
    sent_to_discord BOOLEAN DEFAULT false
);

CREATE TABLE IF NOT EXISTS portfolio_snapshots (
    id SERIAL PRIMARY KEY,
    snapshot_date DATE UNIQUE NOT NULL DEFAULT CURRENT_DATE,
    total_paid NUMERIC NOT NULL,
    total_value NUMERIC NOT NULL,
    total_gain_loss NUMERIC NOT NULL,
    total_gain_loss_pct NUMERIC NOT NULL
);
