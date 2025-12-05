package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	SessionDuration    = 7 * 24 * time.Hour // Sessions last 7 days
	ActivityExtension  = 30 * time.Minute   // Extend session if activity within last 30 min
	SessionTokenLength = 32                 // Length of session token in bytes
)

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateSessionToken creates a cryptographically secure random token
func GenerateSessionToken() (string, error) {
	bytes := make([]byte, SessionTokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// CreateSession creates a new session in the database
func CreateSession(db *sql.DB, playerID int) (string, error) {
	token, err := GenerateSessionToken()
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	expiresAt := time.Now().Add(SessionDuration)

	query := `
		INSERT INTO sessions (player_id, token, last_activity, expires_at)
		VALUES ($1, $2, CURRENT_TIMESTAMP, $3)
	`

	_, err = db.Exec(query, playerID, token, expiresAt)
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}

	return token, nil
}

// ValidateSession checks if session exists and is valid, updates last_activity if needed
func ValidateSession(db *sql.DB, token string) (int, string, error) {
	var playerID int
	var username string
	var lastActivity time.Time
	var expiresAt time.Time

	query := `
		SELECT s.player_id, p.username, s.last_activity, s.expires_at
		FROM sessions s
		JOIN players p ON s.player_id = p.id
		WHERE s.token = $1
	`

	err := db.QueryRow(query, token).Scan(&playerID, &username, &lastActivity, &expiresAt)
	if err == sql.ErrNoRows {
		return 0, "", fmt.Errorf("invalid session")
	}
	if err != nil {
		return 0, "", fmt.Errorf("failed to validate session: %w", err)
	}

	// Check if session has expired
	if time.Now().After(expiresAt) {
		DeleteSession(db, token)
		return 0, "", fmt.Errorf("session expired")
	}

	// Update last_activity if it's been more than ActivityExtension since last update
	// This implements the sliding window - extends expiration on activity
	if time.Since(lastActivity) > ActivityExtension {
		newExpiresAt := time.Now().Add(SessionDuration)
		updateQuery := `
			UPDATE sessions 
			SET last_activity = CURRENT_TIMESTAMP, expires_at = $1
			WHERE token = $2
		`
		_, err = db.Exec(updateQuery, newExpiresAt, token)
		if err != nil {
			// Log error but don't fail the request
			fmt.Printf("Warning: failed to update session activity: %v\n", err)
		}
	}

	return playerID, username, nil
}

// DeleteSession removes a session (logout)
func DeleteSession(db *sql.DB, token string) error {
	query := `DELETE FROM sessions WHERE token = $1`
	_, err := db.Exec(query, token)
	return err
}

// DeletePlayerSessions removes all sessions for a player (logout all devices)
func DeletePlayerSessions(db *sql.DB, playerID int) error {
	query := `DELETE FROM sessions WHERE player_id = $1`
	_, err := db.Exec(query, playerID)
	return err
}

// CleanupExpiredSessions removes expired sessions (run periodically)
func CleanupExpiredSessions(db *sql.DB) error {
	query := `DELETE FROM sessions WHERE expires_at < CURRENT_TIMESTAMP`
	_, err := db.Exec(query)
	return err
}
