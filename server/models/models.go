package models

import (
	"time"
)

type Player struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // Never send password hash in JSON
	Rating       int       `json:"rating"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Game struct {
	ID            int       `json:"id"`
	BlackPlayerID *int      `json:"black_player_id"`
	WhitePlayerID *int      `json:"white_player_id"`
	BoardSize     int       `json:"board_size"`
	Status        string    `json:"status"` // waiting, active, finished
	WinnerID      *int      `json:"winner_id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Move struct {
	ID         int       `json:"id"`
	GameID     int       `json:"game_id"`
	PlayerID   int       `json:"player_id"`
	MoveNumber int       `json:"move_number"`
	X          int       `json:"x"`
	Y          int       `json:"y"`
	CreatedAt  time.Time `json:"created_at"`
}

type CreatePlayerRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

type CreateGameRequest struct {
	BlackPlayerID *int `json:"black_player_id"`
	WhitePlayerID *int `json:"white_player_id"`
	BoardSize     int  `json:"board_size"`
}

type MakeMoveRequest struct {
	GameID   int `json:"game_id"`
	PlayerID int `json:"player_id"`
	X        int `json:"x"`
	Y        int `json:"y"`
}

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token  string `json:"token"`
	Player Player `json:"player"`
}
