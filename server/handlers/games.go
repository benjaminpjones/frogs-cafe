package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"frogs_cafe/middleware"
	"frogs_cafe/models"

	"github.com/go-chi/chi/v5"
)

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

func (h *Handler) ListGames(w http.ResponseWriter, r *http.Request) {
	// Support filtering by status query parameter
	status := r.URL.Query().Get("status")

	var query string
	var args []interface{}

	if status != "" {
		query = "SELECT id, black_player_id, white_player_id, board_size, status, winner_id, creator_id, created_at, updated_at FROM games WHERE status = $1 ORDER BY created_at DESC"
		args = append(args, status)
	} else {
		query = "SELECT id, black_player_id, white_player_id, board_size, status, winner_id, creator_id, created_at, updated_at FROM games ORDER BY created_at DESC"
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Failed to close rows: %v", err)
		}
	}()

	var games []models.Game
	for rows.Next() {
		var g models.Game
		if err := rows.Scan(&g.ID, &g.BlackPlayerID, &g.WhitePlayerID, &g.BoardSize, &g.Status, &g.WinnerID, &g.CreatorID, &g.CreatedAt, &g.UpdatedAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		games = append(games, g)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(games); err != nil {
		log.Printf("Failed to encode games response: %v", err)
	}
}

func (h *Handler) CreateGame(w http.ResponseWriter, r *http.Request) {
	// Get the authenticated player ID from context
	playerID, ok := middleware.GetPlayerID(r)
	if !ok {
		http.Error(w, "Unauthorized: Player ID not found", http.StatusUnauthorized)
		return
	}

	var req models.CreateGameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.BoardSize == 0 {
		req.BoardSize = 19
	}

	// Create game without assigning colors yet - colors will be assigned when someone joins
	// Store creator_id temporarily to track who created the game
	var game models.Game
	err := h.db.QueryRow(
		"INSERT INTO games (black_player_id, white_player_id, board_size, status, creator_id) VALUES (NULL, NULL, $1, 'waiting', $2) RETURNING id, black_player_id, white_player_id, board_size, status, winner_id, creator_id, created_at, updated_at",
		req.BoardSize, playerID,
	).Scan(&game.ID, &game.BlackPlayerID, &game.WhitePlayerID, &game.BoardSize, &game.Status, &game.WinnerID, &game.CreatorID, &game.CreatedAt, &game.UpdatedAt)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(game); err != nil {
		log.Printf("Failed to encode game response: %v", err)
	}
}

func (h *Handler) JoinGame(w http.ResponseWriter, r *http.Request) {
	// Get the authenticated player ID from context
	joinerID, ok := middleware.GetPlayerID(r)
	if !ok {
		http.Error(w, "Unauthorized: Player ID not found", http.StatusUnauthorized)
		return
	}

	gameID := chi.URLParam(r, "gameID")
	id, err := strconv.Atoi(gameID)
	if err != nil {
		http.Error(w, "Invalid game ID", http.StatusBadRequest)
		return
	}

	// Get the current game state with creator_id
	var game models.Game
	var creatorID int
	err = h.db.QueryRow(
		"SELECT id, black_player_id, white_player_id, board_size, status, winner_id, creator_id, created_at, updated_at FROM games WHERE id = $1",
		id,
	).Scan(&game.ID, &game.BlackPlayerID, &game.WhitePlayerID, &game.BoardSize, &game.Status, &game.WinnerID, &creatorID, &game.CreatedAt, &game.UpdatedAt)

	if err == sql.ErrNoRows {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Validate game can be joined
	if game.Status != "waiting" {
		http.Error(w, "Game is not available to join", http.StatusBadRequest)
		return
	}

	if creatorID == joinerID {
		http.Error(w, "Cannot join your own game", http.StatusBadRequest)
		return
	}

	if game.BlackPlayerID != nil || game.WhitePlayerID != nil {
		http.Error(w, "Game already has two players", http.StatusBadRequest)
		return
	}

	// Get ratings for both players to determine colors
	var creatorRating, joinerRating int
	err = h.db.QueryRow("SELECT rating FROM players WHERE id = $1", creatorID).Scan(&creatorRating)
	if err != nil {
		http.Error(w, "Failed to get creator rating", http.StatusInternalServerError)
		return
	}

	err = h.db.QueryRow("SELECT rating FROM players WHERE id = $1", joinerID).Scan(&joinerRating)
	if err != nil {
		http.Error(w, "Failed to get joiner rating", http.StatusInternalServerError)
		return
	}

	// If ratings are within 50 points, randomly assign
	var blackPlayerID, whitePlayerID int
	ratingDiff := creatorRating - joinerRating

	if ratingDiff < -50 {
		// Creator is weaker, gets black
		blackPlayerID = creatorID
		whitePlayerID = joinerID
	} else if ratingDiff > 50 {
		// Joiner is weaker, gets black
		blackPlayerID = joinerID
		whitePlayerID = creatorID
	} else {
		// Ratings are close, randomly assign (based on game ID for determinism)
		if id%2 == 0 {
			blackPlayerID = creatorID
			whitePlayerID = joinerID
		} else {
			blackPlayerID = joinerID
			whitePlayerID = creatorID
		}
	}

	// Update game with both players and set status to active
	err = h.db.QueryRow(
		"UPDATE games SET black_player_id = $1, white_player_id = $2, status = 'active', updated_at = CURRENT_TIMESTAMP WHERE id = $3 RETURNING id, black_player_id, white_player_id, board_size, status, winner_id, creator_id, created_at, updated_at",
		blackPlayerID, whitePlayerID, id,
	).Scan(&game.ID, &game.BlackPlayerID, &game.WhitePlayerID, &game.BoardSize, &game.Status, &game.WinnerID, &game.CreatorID, &game.CreatedAt, &game.UpdatedAt)

	if err != nil {
		log.Printf("Failed to update game status: %v", err)
		http.Error(w, "Failed to join game", http.StatusInternalServerError)
		return
	}

	log.Printf("Game #%d status changed to 'active' - Black: Player #%d, White: Player #%d", game.ID, blackPlayerID, whitePlayerID)

	// Broadcast game status update via WebSocket
	if hub := GetHub(); hub != nil {
		gameUpdate := map[string]interface{}{
			"type": "game_update",
			"data": map[string]interface{}{
				"game_id":         game.ID,
				"status":          game.Status,
				"black_player_id": game.BlackPlayerID,
				"white_player_id": game.WhitePlayerID,
				"game":            game,
			},
		}
		if updateBytes, err := json.Marshal(gameUpdate); err == nil {
			hub.broadcast <- updateBytes
		} else {
			log.Printf("Failed to marshal game_update message: %v", err)
		}
	} else {
		log.Printf("Warning: WebSocket hub not initialized, skipping broadcast")
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(game); err != nil {
		log.Printf("Failed to encode game response: %v", err)
	}
}

func (h *Handler) GetGame(w http.ResponseWriter, r *http.Request) {
	gameID := chi.URLParam(r, "gameID")
	id, err := strconv.Atoi(gameID)
	if err != nil {
		http.Error(w, "Invalid game ID", http.StatusBadRequest)
		return
	}

	var game models.Game
	err = h.db.QueryRow(
		"SELECT id, black_player_id, white_player_id, board_size, status, winner_id, creator_id, created_at, updated_at FROM games WHERE id = $1",
		id,
	).Scan(&game.ID, &game.BlackPlayerID, &game.WhitePlayerID, &game.BoardSize, &game.Status, &game.WinnerID, &game.CreatorID, &game.CreatedAt, &game.UpdatedAt)

	if err == sql.ErrNoRows {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(game); err != nil {
		log.Printf("Failed to encode game response: %v", err)
	}
}

func (h *Handler) GetGameMoves(w http.ResponseWriter, r *http.Request) {
	gameID := chi.URLParam(r, "gameID")
	id, err := strconv.Atoi(gameID)
	if err != nil {
		http.Error(w, "Invalid game ID", http.StatusBadRequest)
		return
	}

	rows, err := h.db.Query(
		"SELECT id, game_id, player_id, move_number, x, y, created_at FROM moves WHERE game_id = $1 ORDER BY move_number ASC",
		id,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Failed to close rows: %v", err)
		}
	}()

	// Initialize as empty slice to ensure JSON encoding returns [] instead of null
	moves := []models.Move{}
	for rows.Next() {
		var m models.Move
		if err := rows.Scan(&m.ID, &m.GameID, &m.PlayerID, &m.MoveNumber, &m.X, &m.Y, &m.CreatedAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		moves = append(moves, m)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(moves); err != nil {
		log.Printf("Failed to encode moves response: %v", err)
	}
}
