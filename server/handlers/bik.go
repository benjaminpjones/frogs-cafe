package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"frogs_cafe/bik"
	"frogs_cafe/middleware"

	"github.com/go-chi/chi/v5"
)

// WellKnownWebFinger handles GET /.well-known/webfinger
func (h *Handler) WellKnownWebFinger(w http.ResponseWriter, r *http.Request) {
	resource := r.URL.Query().Get("resource")
	// Expect: acct:username@domain
	if !strings.HasPrefix(resource, "acct:") {
		http.Error(w, "unsupported resource type", http.StatusBadRequest)
		return
	}
	acct := strings.TrimPrefix(resource, "acct:")
	atIdx := strings.LastIndex(acct, "@")
	if atIdx < 0 {
		http.Error(w, "invalid acct", http.StatusBadRequest)
		return
	}
	username := acct[:atIdx]

	var id int
	err := h.db.QueryRow("SELECT id FROM players WHERE username = $1", username).Scan(&id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	resp := bik.WebFingerResponse{
		Subject: resource,
		Links: []bik.WebFingerLink{{
			Rel:  "self",
			Type: "application/activity+json",
			Href: fmt.Sprintf("%s/users/%s", h.cfg.BaseURL, username),
		}},
	}
	w.Header().Set("Content-Type", "application/jrd+json")
	json.NewEncoder(w).Encode(resp)
}

// WellKnownBIKKeys handles GET /.well-known/bik/keys
func (h *Handler) WellKnownBIKKeys(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.bikKeys.JWKSet())
}

// BIKInbox handles POST /bik/inbox — receives AP activities from remote servers
func (h *Handler) BIKInbox(w http.ResponseWriter, r *http.Request) {
	var activity bik.Activity
	if err := json.NewDecoder(r.Body).Decode(&activity); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	switch activity.Type {
	case "Accept":
		h.handleChallengeAccept(w, r, activity)
	default:
		http.Error(w, "unsupported activity type", http.StatusNotImplemented)
	}
}

func (h *Handler) handleChallengeAccept(w http.ResponseWriter, r *http.Request, activity bik.Activity) {
	challengeURI, ok := activity.Object.(string)
	if !ok {
		http.Error(w, "object must be a challenge URI string", http.StatusBadRequest)
		return
	}

	// Load the challenge
	var (
		challengeID int
		creatorID   int
		boardSize   int
		timeCtrlRaw []byte
		status      string
		expiresAt   time.Time
	)
	err := h.db.QueryRow(`
		SELECT id, creator_id, board_size, time_control, status, expires_at
		FROM bik_challenges WHERE uri = $1
	`, challengeURI).Scan(&challengeID, &creatorID, &boardSize, &timeCtrlRaw, &status, &expiresAt)
	if err != nil {
		http.Error(w, "challenge not found", http.StatusNotFound)
		return
	}
	if status != "open" {
		http.Error(w, "challenge no longer open", http.StatusConflict)
		return
	}
	if time.Now().After(expiresAt) {
		http.Error(w, "challenge expired", http.StatusGone)
		return
	}

	// Look up the host player's username
	var creatorUsername string
	h.db.QueryRow("SELECT username FROM players WHERE id = $1", creatorID).Scan(&creatorUsername)

	// Assign colors: creator gets black for now (TODO: respect colorPref)
	blackActor := fmt.Sprintf("%s/users/%s", h.cfg.BaseURL, creatorUsername)
	whiteActor := activity.Actor

	// Create the game
	gameID := fmt.Sprintf("%d", time.Now().UnixNano()) // simple unique ID
	var dbGameID int
	err = h.db.QueryRow(`
		INSERT INTO games (black_player_id, board_size, status, creator_id, black_actor_uri, white_actor_uri, bik_challenge_uri)
		VALUES ($1, $2, 'active', $1, $3, $4, $5)
		RETURNING id
	`, creatorID, boardSize, blackActor, whiteActor, challengeURI).Scan(&dbGameID)
	if err != nil {
		http.Error(w, "failed to create game", http.StatusInternalServerError)
		return
	}
	gameID = fmt.Sprintf("%d", dbGameID)

	// Mark challenge as accepted
	h.db.Exec("UPDATE bik_challenges SET status = 'accepted' WHERE id = $1", challengeID)

	// Build response: CreateGame activity
	gameURI := fmt.Sprintf("%s/games/%s", h.cfg.BaseURL, gameID)
	wsURL := fmt.Sprintf("%s/ws/games/%s", strings.Replace(h.cfg.BaseURL, "http", "ws", 1), gameID)
	gameActivity := bik.Activity{
		Context: bik.APContext,
		Type:    "Create",
		Actor:   blackActor,
		Object: bik.NoteObject{
			Type:      "Note",
			ID:        gameURI,
			Content:   fmt.Sprintf("Game started! Watch at %s", gameURI),
			InReplyTo: challengeURI,
			Attachment: bik.BikGame{
				Type:      "BikGame",
				ID:        gameID,
				Black:     blackActor,
				White:     whiteActor,
				URL:       gameURI,
				WebSocket: wsURL,
			},
		},
	}

	// Also broadcast Undo for the challenge so followers know it's taken
	go h.broadcastUndoChallenge(challengeURI, blackActor)

	w.Header().Set("Content-Type", "application/activity+json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(gameActivity)
}

func (h *Handler) broadcastUndoChallenge(challengeURI, actor string) {
	// TODO: deliver Undo activity to followers
	// For now just log — full AP delivery is a future extension
	_ = challengeURI
	_ = actor
}

// CreateBIKChallenge handles POST /api/v1/bik/challenges
func (h *Handler) CreateBIKChallenge(w http.ResponseWriter, r *http.Request) {
	playerID, _ := middleware.GetPlayerID(r)

	var req struct {
		BoardSize   int             `json:"boardSize"`
		TimeControl bik.TimeControl `json:"timeControl"`
		ColorPref   string          `json:"colorPreference"`
		ExpiresIn   int             `json:"expiresIn"` // seconds
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if req.BoardSize == 0 {
		req.BoardSize = 19
	}
	if req.ExpiresIn == 0 {
		req.ExpiresIn = 3600 // 1 hour default
	}
	if req.ColorPref == "" {
		req.ColorPref = "any"
	}

	var username string
	h.db.QueryRow("SELECT username FROM players WHERE id = $1", playerID).Scan(&username)

	challengeID := fmt.Sprintf("%d", time.Now().UnixNano())
	challengeURI := fmt.Sprintf("%s/challenges/%s", h.cfg.BaseURL, challengeID)
	expiresAt := time.Now().Add(time.Duration(req.ExpiresIn) * time.Second)

	timeCtrlJSON, _ := json.Marshal(req.TimeControl)
	_, err := h.db.Exec(`
		INSERT INTO bik_challenges (creator_id, uri, board_size, time_control, color_pref, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, playerID, challengeURI, req.BoardSize, timeCtrlJSON, req.ColorPref, expiresAt)
	if err != nil {
		http.Error(w, "failed to create challenge", http.StatusInternalServerError)
		return
	}

	actorURI := fmt.Sprintf("%s/users/%s", h.cfg.BaseURL, username)
	content := fmt.Sprintf("Looking for a game! %dx%d", req.BoardSize, req.BoardSize)

	activity := bik.Activity{
		Context: bik.APContext,
		Type:    "Create",
		Actor:   actorURI,
		Object: bik.NoteObject{
			Type:         "Note",
			ID:           challengeURI,
			AttributedTo: actorURI,
			Content:      content,
			Attachment: bik.BikChallenge{
				Type:        "BikChallenge",
				BoardSize:   req.BoardSize,
				TimeControl: req.TimeControl,
				ColorPref:   req.ColorPref,
				ExpiresAt:   expiresAt,
			},
		},
	}

	w.Header().Set("Content-Type", "application/activity+json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(activity)
}

// ListBIKChallenges handles GET /api/v1/bik/challenges
func (h *Handler) ListBIKChallenges(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(`
		SELECT c.uri, p.username, c.board_size, c.time_control, c.color_pref, c.expires_at
		FROM bik_challenges c
		JOIN players p ON p.id = c.creator_id
		WHERE c.status = 'open' AND c.expires_at > NOW()
		ORDER BY c.created_at DESC
	`)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type challengeItem struct {
		URI       string          `json:"uri"`
		Creator   string          `json:"creator"`
		BoardSize int             `json:"boardSize"`
		TimeCtrl  json.RawMessage `json:"timeControl"`
		ColorPref string          `json:"colorPreference"`
		ExpiresAt time.Time       `json:"expiresAt"`
	}

	items := []challengeItem{}
	for rows.Next() {
		var item challengeItem
		var timeCtrlRaw []byte
		rows.Scan(&item.URI, &item.Creator, &item.BoardSize, &timeCtrlRaw, &item.ColorPref, &item.ExpiresAt)
		item.TimeCtrl = json.RawMessage(timeCtrlRaw)
		items = append(items, item)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

// GetBIKToken handles POST /api/v1/bik/token — issues a signed token for a remote game
func (h *Handler) GetBIKToken(w http.ResponseWriter, r *http.Request) {
	playerID, _ := middleware.GetPlayerID(r)
	gameID := chi.URLParam(r, "gameID")

	var username string
	h.db.QueryRow("SELECT username FROM players WHERE id = $1", playerID).Scan(&username)

	// Allow caller to provide the full game URI for cross-server games.
	// Falls back to this server's own game URI if not specified.
	gameURI := r.URL.Query().Get("gameURI")
	if gameURI == "" {
		gameURI = fmt.Sprintf("%s/games/%s", h.cfg.BaseURL, gameID)
	}
	actorURI := fmt.Sprintf("%s/users/%s", h.cfg.BaseURL, username)

	token, err := h.bikKeys.SignToken(actorURI, gameURI, 5*time.Minute)
	if err != nil {
		http.Error(w, "failed to sign token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}
