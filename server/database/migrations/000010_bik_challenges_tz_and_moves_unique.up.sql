-- Fix bik_challenges timestamps to use TIMESTAMPTZ
ALTER TABLE bik_challenges ALTER COLUMN expires_at TYPE TIMESTAMPTZ;
ALTER TABLE bik_challenges ALTER COLUMN created_at TYPE TIMESTAMPTZ;

-- Unique constraint on moves to prevent race conditions
ALTER TABLE moves ADD CONSTRAINT moves_game_id_move_number_unique UNIQUE (game_id, move_number);
