package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"frogs_cafe/database"
	"frogs_cafe/models"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	db *database.DB
}

func New(db *database.DB) *Handler {
	return &Handler{db: db}
}

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// Player handlers
func (h *Handler) ListPlayers(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query("SELECT id, username, email, rating, created_at, updated_at FROM players ORDER BY rating DESC")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var players []models.Player
	for rows.Next() {
		var p models.Player
		if err := rows.Scan(&p.ID, &p.Username, &p.Email, &p.Rating, &p.CreatedAt, &p.UpdatedAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		players = append(players, p)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(players)
}

func (h *Handler) CreatePlayer(w http.ResponseWriter, r *http.Request) {
	var req models.CreatePlayerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var player models.Player
	err := h.db.QueryRow(
		"INSERT INTO players (username, email) VALUES ($1, $2) RETURNING id, username, email, rating, created_at, updated_at",
		req.Username, req.Email,
	).Scan(&player.ID, &player.Username, &player.Email, &player.Rating, &player.CreatedAt, &player.UpdatedAt)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(player)
}

func (h *Handler) GetPlayer(w http.ResponseWriter, r *http.Request) {
	playerID := chi.URLParam(r, "playerID")
	id, err := strconv.Atoi(playerID)
	if err != nil {
		http.Error(w, "Invalid player ID", http.StatusBadRequest)
		return
	}

	var player models.Player
	err = h.db.QueryRow(
		"SELECT id, username, email, rating, created_at, updated_at FROM players WHERE id = $1",
		id,
	).Scan(&player.ID, &player.Username, &player.Email, &player.Rating, &player.CreatedAt, &player.UpdatedAt)

	if err == sql.ErrNoRows {
		http.Error(w, "Player not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(player)
}

// Game handlers
func (h *Handler) ListGames(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query("SELECT id, black_player_id, white_player_id, board_size, status, winner_id, created_at, updated_at FROM games ORDER BY created_at DESC")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var games []models.Game
	for rows.Next() {
		var g models.Game
		if err := rows.Scan(&g.ID, &g.BlackPlayerID, &g.WhitePlayerID, &g.BoardSize, &g.Status, &g.WinnerID, &g.CreatedAt, &g.UpdatedAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		games = append(games, g)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(games)
}

func (h *Handler) CreateGame(w http.ResponseWriter, r *http.Request) {
	var req models.CreateGameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.BoardSize == 0 {
		req.BoardSize = 19
	}

	var game models.Game
	err := h.db.QueryRow(
		"INSERT INTO games (black_player_id, white_player_id, board_size) VALUES ($1, $2, $3) RETURNING id, black_player_id, white_player_id, board_size, status, winner_id, created_at, updated_at",
		req.BlackPlayerID, req.WhitePlayerID, req.BoardSize,
	).Scan(&game.ID, &game.BlackPlayerID, &game.WhitePlayerID, &game.BoardSize, &game.Status, &game.WinnerID, &game.CreatedAt, &game.UpdatedAt)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(game)
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
		"SELECT id, black_player_id, white_player_id, board_size, status, winner_id, created_at, updated_at FROM games WHERE id = $1",
		id,
	).Scan(&game.ID, &game.BlackPlayerID, &game.WhitePlayerID, &game.BoardSize, &game.Status, &game.WinnerID, &game.CreatedAt, &game.UpdatedAt)

	if err == sql.ErrNoRows {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(game)
}
