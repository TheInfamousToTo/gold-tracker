-- One row per calendar month (keyed by the month's first day). Rerunning a
-- report for the same month replaces it. Idempotent on every boot.
CREATE TABLE IF NOT EXISTS monthly_reports (
    month DATE PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    data JSONB NOT NULL,
    analysis TEXT,
    model TEXT,
    error TEXT,
    discord_sent_at TIMESTAMP
);
