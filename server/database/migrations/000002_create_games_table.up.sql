CREATE TABLE IF NOT EXISTS games (
	id SERIAL PRIMARY KEY,
	black_player_id INTEGER REFERENCES players(id),
	white_player_id INTEGER REFERENCES players(id),
	board_size INTEGER DEFAULT 19,
	status VARCHAR(50) DEFAULT 'waiting',
	winner_id INTEGER REFERENCES players(id),
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_games_status ON games(status);
