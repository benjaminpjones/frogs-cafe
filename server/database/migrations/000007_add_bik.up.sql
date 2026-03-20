-- BIK challenges created by local players
CREATE TABLE IF NOT EXISTS bik_challenges (
    id          SERIAL PRIMARY KEY,
    creator_id  INTEGER NOT NULL REFERENCES players(id),
    uri         TEXT NOT NULL UNIQUE,
    board_size  INTEGER NOT NULL DEFAULT 19,
    time_control JSONB NOT NULL,
    color_pref  VARCHAR(10) NOT NULL DEFAULT 'any',
    status      VARCHAR(20) NOT NULL DEFAULT 'open', -- open, accepted, cancelled
    expires_at  TIMESTAMP NOT NULL,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Allow games to have remote (federated) players alongside local ones
ALTER TABLE games
    ADD COLUMN IF NOT EXISTS black_actor_uri TEXT,
    ADD COLUMN IF NOT EXISTS white_actor_uri TEXT,
    ADD COLUMN IF NOT EXISTS bik_challenge_uri TEXT;
