package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"frogs_cafe/auth"
	"frogs_cafe/models"
)

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	var player models.Player
	err = h.db.QueryRow(
		"INSERT INTO players (username, email, password_hash) VALUES ($1, $2, $3) RETURNING id, username, email, rating, created_at, updated_at",
		req.Username, req.Email, hashedPassword,
	).Scan(&player.ID, &player.Username, &player.Email, &player.Rating, &player.CreatedAt, &player.UpdatedAt)

	if err != nil {
		http.Error(w, "Username or email already exists", http.StatusConflict)
		return
	}

	token, err := auth.CreateSession(h.db.DB, player.ID)
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(models.AuthResponse{
		Token:  token,
		Player: player,
	}); err != nil {
		log.Printf("Failed to encode auth response: %v", err)
	}
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

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

	if !auth.CheckPasswordHash(req.Password, player.PasswordHash) {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	token, err := auth.CreateSession(h.db.DB, player.ID)
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(models.AuthResponse{
		Token:  token,
		Player: player,
	}); err != nil {
		log.Printf("Failed to encode auth response: %v", err)
	}
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
