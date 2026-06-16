#!/bin/bash
set -e

echo "=== Gold Tracker Database Setup ==="
read -p "PostgreSQL host [localhost]: " PGHOST
PGHOST=${PGHOST:-localhost}

read -p "PostgreSQL port [5432]: " PGPORT
PGPORT=${PGPORT:-5432}

read -p "Postgres superuser [postgres]: " PGSUPERUSER
PGSUPERUSER=${PGSUPERUSER:-postgres}

read -p "New DB name [gold_tracker]: " DBNAME
DBNAME=${DBNAME:-gold_tracker}

read -p "New app username [gold_admin]: " DBUSER
DBUSER=${DBUSER:-gold_admin}

read -sp "Enter password for superuser ($PGSUPERUSER): " PGSUPERPASS
echo
read -sp "Enter password for app user ($DBUSER): " DBUSERPASS
echo
echo "---"

# Create database + role using the superuser connection
export PGPASSWORD="$PGSUPERPASS"

echo "Creating database and role..."
psql -h "$PGHOST" -p "$PGPORT" -U "$PGSUPERUSER" -d postgres -c "CREATE DATABASE $DBNAME;" 2>/dev/null || echo "  - Database already exists, skipping."
psql -h "$PGHOST" -p "$PGPORT" -U "$PGSUPERUSER" -d postgres -c "CREATE USER $DBUSER WITH ENCRYPTED PASSWORD '$DBUSERPASS';" 2>/dev/null || echo "  - Role already exists, updating password."
psql -h "$PGHOST" -p "$PGPORT" -U "$PGSUPERUSER" -d postgres -c "ALTER USER $DBUSER WITH PASSWORD '$DBUSERPASS';"
psql -h "$PGHOST" -p "$PGPORT" -U "$PGSUPERUSER" -d postgres -c "GRANT ALL PRIVILEGES ON DATABASE $DBNAME TO $DBUSER;"

echo "Granting schema privileges inside '$DBNAME'..."
psql -h "$PGHOST" -p "$PGPORT" -U "$PGSUPERUSER" -d "$DBNAME" -c "GRANT ALL ON SCHEMA public TO $DBUSER;"

# Build the schema as the app user
export PGPASSWORD="$DBUSERPASS"

echo "Creating tables and view..."
psql -h "$PGHOST" -p "$PGPORT" -U "$DBUSER" -d "$DBNAME" <<'EOF'
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
EOF

unset PGPASSWORD

echo "=== Done. Database '$DBNAME' is ready for user '$DBUSER'. ==="
