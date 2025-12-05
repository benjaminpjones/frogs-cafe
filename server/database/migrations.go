package database

import (
	"fmt"
)

func RunMigrations(db *DB) error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS players (
			id SERIAL PRIMARY KEY,
			username VARCHAR(255) UNIQUE NOT NULL,
			email VARCHAR(255) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			rating INTEGER DEFAULT 1500,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS games (
			id SERIAL PRIMARY KEY,
			black_player_id INTEGER REFERENCES players(id),
			white_player_id INTEGER REFERENCES players(id),
			board_size INTEGER DEFAULT 19,
			status VARCHAR(50) DEFAULT 'waiting',
			winner_id INTEGER REFERENCES players(id),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS moves (
			id SERIAL PRIMARY KEY,
			game_id INTEGER REFERENCES games(id) ON DELETE CASCADE,
			player_id INTEGER REFERENCES players(id),
			move_number INTEGER NOT NULL,
			x INTEGER NOT NULL,
			y INTEGER NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_games_status ON games(status)`,
		`CREATE INDEX IF NOT EXISTS idx_moves_game_id ON moves(game_id)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			id SERIAL PRIMARY KEY,
			player_id INTEGER NOT NULL REFERENCES players(id) ON DELETE CASCADE,
			token VARCHAR(255) UNIQUE NOT NULL,
			last_activity TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			expires_at TIMESTAMP NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_player_id ON sessions(player_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at)`,
	}

	for i, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("migration %d failed: %w", i+1, err)
		}
	}

	return nil
}
