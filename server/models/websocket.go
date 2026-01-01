package models

// WebSocketMessage represents the structure of messages sent over WebSocket
type WebSocketMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// MoveData represents the data payload for a move message
type MoveData struct {
	X        int `json:"x"`
	Y        int `json:"y"`
	GameID   int `json:"game_id"`
	PlayerID int `json:"player_id"`
}

// GameUpdateData represents the data payload for a game status update
type GameUpdateData struct {
	GameID        int    `json:"game_id"`
	Status        string `json:"status"`
	BlackPlayerID *int   `json:"black_player_id"`
	WhitePlayerID *int   `json:"white_player_id"`
	Game          *Game  `json:"game"`
}
