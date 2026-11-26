-- How far each metal's daily value has been posted to Sure, so a restart
-- resumes instead of re-sending the whole history.
CREATE TABLE IF NOT EXISTS sure_sync (
    metal TEXT PRIMARY KEY,
    synced_through DATE NOT NULL
);
