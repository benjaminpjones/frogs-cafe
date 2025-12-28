package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"frogs_cafe/models"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) ListPlayers(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query("SELECT id, username, email, rating, created_at, updated_at FROM players ORDER BY rating DESC")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Failed to close rows: %v", err)
		}
	}()

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
	if err := json.NewEncoder(w).Encode(players); err != nil {
		log.Printf("Failed to encode players response: %v", err)
	}
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
	if err := json.NewEncoder(w).Encode(player); err != nil {
		log.Printf("Failed to encode player response: %v", err)
	}
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
	if err := json.NewEncoder(w).Encode(player); err != nil {
		log.Printf("Failed to encode player response: %v", err)
	}
}
