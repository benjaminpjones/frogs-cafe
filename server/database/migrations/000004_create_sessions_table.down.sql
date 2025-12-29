DROP INDEX IF EXISTS idx_sessions_expires_at;
DROP INDEX IF EXISTS idx_sessions_player_id;
DROP INDEX IF EXISTS idx_sessions_token;
DROP TABLE IF EXISTS sessions CASCADE;
