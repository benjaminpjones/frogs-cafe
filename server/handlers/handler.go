package handlers

import (
	"strconv"

	"frogs_cafe/database"
)

type Handler struct {
	db *database.DB
}

func New(db *database.DB) *Handler {
	h := &Handler{db: db}
	InitHub(h)
	return h
}

func (h *Handler) SaveMove(gameIDStr string, playerID int, data map[string]interface{}) error {
	gameID, err := strconv.Atoi(gameIDStr)
	if err != nil {
		return err
	}

	x := int(data["x"].(float64))
	y := int(data["y"].(float64))

	// Get the current move number
	var moveNumber int
	err = h.db.QueryRow(
		"SELECT COALESCE(MAX(move_number), 0) + 1 FROM moves WHERE game_id = $1",
		gameID,
	).Scan(&moveNumber)
	if err != nil {
		return err
	}

	// Insert the move
	_, err = h.db.Exec(
		"INSERT INTO moves (game_id, player_id, move_number, x, y) VALUES ($1, $2, $3, $4, $5)",
		gameID, playerID, moveNumber, x, y,
	)
	return err
}
