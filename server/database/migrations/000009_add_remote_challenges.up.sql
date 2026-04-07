CREATE TABLE IF NOT EXISTS remote_challenges (
    id          SERIAL PRIMARY KEY,
    uri         TEXT NOT NULL UNIQUE,
    actor_uri   TEXT NOT NULL,
    board_size  INTEGER NOT NULL DEFAULT 19,
    time_control JSONB,
    color_assignment VARCHAR(10) NOT NULL DEFAULT 'random',
    inbox_url   TEXT NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    received_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);
