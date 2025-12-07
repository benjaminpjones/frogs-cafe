package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"frogs_cafe/auth"
	"frogs_cafe/database"
	"frogs_cafe/middleware"
	"frogs_cafe/models"

	"github.com/go-chi/chi/v5"
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
	defer rows.Close()

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
	json.NewEncoder(w).Encode(games)
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
	json.NewEncoder(w).Encode(game)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
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
	json.NewEncoder(w).Encode(game)
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
	defer rows.Close()

	var moves []models.Move
	for rows.Next() {
		var m models.Move
		if err := rows.Scan(&m.ID, &m.GameID, &m.PlayerID, &m.MoveNumber, &m.X, &m.Y, &m.CreatedAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		moves = append(moves, m)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(moves)
}

// Auth handlers
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate input
	if req.Username == "" || req.Email == "" || req.Password == "" {
		http.Error(w, "Username, email, and password are required", http.StatusBadRequest)
		return
	}

	if len(req.Password) < 6 {
		http.Error(w, "Password must be at least 6 characters", http.StatusBadRequest)
		return
	}

	// Hash password
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	// Create player
	var player models.Player
	err = h.db.QueryRow(
		"INSERT INTO players (username, email, password_hash) VALUES ($1, $2, $3) RETURNING id, username, email, rating, created_at, updated_at",
		req.Username, req.Email, hashedPassword,
	).Scan(&player.ID, &player.Username, &player.Email, &player.Rating, &player.CreatedAt, &player.UpdatedAt)

	if err != nil {
		// Check for unique constraint violations
		http.Error(w, "Username or email already exists", http.StatusConflict)
		return
	}

	// Create session
	token, err := auth.CreateSession(h.db.DB, player.ID)
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(models.AuthResponse{
		Token:  token,
		Player: player,
	})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate input
	if req.Username == "" || req.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	// Get player from database
	var player models.Player
	err := h.db.QueryRow(
		"SELECT id, username, email, password_hash, rating, created_at, updated_at FROM players WHERE username = $1",
		req.Username,
	).Scan(&player.ID, &player.Username, &player.Email, &player.PasswordHash, &player.Rating, &player.CreatedAt, &player.UpdatedAt)

	if err == sql.ErrNoRows {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Check password
	if !auth.CheckPasswordHash(req.Password, player.PasswordHash) {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	// Create session
	token, err := auth.CreateSession(h.db.DB, player.ID)
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.AuthResponse{
		Token:  token,
		Player: player,
	})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	token := r.URL.Query().Get("token")
	
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		token = strings.TrimPrefix(authHeader, "Bearer ")
	}

	if token == "" {
		http.Error(w, "No token provided", http.StatusUnauthorized)
		return
	}

	if err := auth.DeleteSession(h.db.DB, token); err != nil {
		http.Error(w, "Failed to logout", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
