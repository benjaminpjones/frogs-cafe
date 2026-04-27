ALTER TABLE moves DROP CONSTRAINT IF EXISTS moves_game_id_move_number_unique;
ALTER TABLE bik_challenges ALTER COLUMN expires_at TYPE TIMESTAMP;
ALTER TABLE bik_challenges ALTER COLUMN created_at TYPE TIMESTAMP;
