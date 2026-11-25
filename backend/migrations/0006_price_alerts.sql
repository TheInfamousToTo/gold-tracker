-- Last big-drop alert sent per metal, so a slide is reported once
-- rather than every hour.
CREATE TABLE IF NOT EXISTS price_alerts (
    metal TEXT PRIMARY KEY,
    alerted_on DATE NOT NULL,
    price NUMERIC(12, 4) NOT NULL
);
