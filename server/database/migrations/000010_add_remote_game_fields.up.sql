-- For games hosted on remote servers, store the remote game URI and WebSocket URL
ALTER TABLE games ADD COLUMN IF NOT EXISTS remote_game_uri TEXT;
ALTER TABLE games ADD COLUMN IF NOT EXISTS remote_ws_url TEXT;

-- Fix bik_challenges timestamps to use TIMESTAMPTZ
ALTER TABLE bik_challenges ALTER COLUMN expires_at TYPE TIMESTAMPTZ;
ALTER TABLE bik_challenges ALTER COLUMN created_at TYPE TIMESTAMPTZ;

-- Unique constraint on moves to prevent race conditions
ALTER TABLE moves ADD CONSTRAINT moves_game_id_move_number_unique UNIQUE (game_id, move_number);
