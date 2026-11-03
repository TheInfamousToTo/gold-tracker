-- Stocks and ETFs are separate from physical metals, which keeps legacy
-- metal purity and price-feed assumptions intact. Idempotent on every boot.
CREATE TABLE IF NOT EXISTS stock_brokers (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    buy_percent NUMERIC NOT NULL DEFAULT 0 CHECK (buy_percent BETWEEN 0 AND 100),
    sell_percent NUMERIC NOT NULL DEFAULT 0 CHECK (sell_percent BETWEEN 0 AND 100),
    fixed_fee NUMERIC NOT NULL DEFAULT 0 CHECK (fixed_fee >= 0),
    minimum_fee NUMERIC NOT NULL DEFAULT 0 CHECK (minimum_fee >= 0),
    tax_percent NUMERIC NOT NULL DEFAULT 0 CHECK (tax_percent BETWEEN 0 AND 100),
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS stock_positions (
    symbol TEXT PRIMARY KEY,
    asset_name TEXT NOT NULL,
    currency TEXT NOT NULL,
    shares NUMERIC NOT NULL DEFAULT 0 CHECK (shares >= 0),
    cost_basis_bhd NUMERIC NOT NULL DEFAULT 0 CHECK (cost_basis_bhd >= 0),
    realized_pnl_bhd NUMERIC NOT NULL DEFAULT 0,
    latest_price NUMERIC CHECK (latest_price > 0),
    latest_fx_to_bhd NUMERIC CHECK (latest_fx_to_bhd > 0),
    latest_price_date DATE,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    CHECK ((latest_price IS NULL AND latest_fx_to_bhd IS NULL AND latest_price_date IS NULL) OR
           (latest_price IS NOT NULL AND latest_fx_to_bhd IS NOT NULL AND latest_price_date IS NOT NULL))
);

CREATE TABLE IF NOT EXISTS stock_trades (
    id BIGSERIAL PRIMARY KEY,
    symbol TEXT NOT NULL REFERENCES stock_positions(symbol),
    broker_id INTEGER REFERENCES stock_brokers(id) ON DELETE SET NULL,
    broker_name TEXT,
    side TEXT NOT NULL CHECK (side IN ('buy', 'sell')),
    trade_date DATE NOT NULL,
    shares NUMERIC NOT NULL CHECK (shares > 0),
    unit_price NUMERIC NOT NULL CHECK (unit_price > 0),
    currency TEXT NOT NULL,
    fx_to_bhd NUMERIC NOT NULL CHECK (fx_to_bhd > 0),
    notional_bhd NUMERIC NOT NULL CHECK (notional_bhd >= 0),
    fees_bhd NUMERIC NOT NULL CHECK (fees_bhd >= 0),
    total_bhd NUMERIC NOT NULL,
    realized_pnl_bhd NUMERIC NOT NULL DEFAULT 0,
    notes TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_stock_trades_symbol_date
    ON stock_trades (symbol, trade_date DESC, id DESC);
