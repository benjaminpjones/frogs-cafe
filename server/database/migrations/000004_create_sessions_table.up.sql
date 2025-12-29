CREATE TABLE IF NOT EXISTS sessions (
	id SERIAL PRIMARY KEY,
	player_id INTEGER NOT NULL REFERENCES players(id) ON DELETE CASCADE,
	token VARCHAR(255) UNIQUE NOT NULL,
	last_activity TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	expires_at TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token);
CREATE INDEX IF NOT EXISTS idx_sessions_player_id ON sessions(player_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);
