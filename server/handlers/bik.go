package handlers

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	mathrand "math/rand"
	"net/http"
	"net/url"
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
	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	} else if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
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
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("WellKnownWebFinger: encode: %v", err)
	}
}

// ActorDocument handles GET /users/{username} — returns an AP Person object
func (h *Handler) ActorDocument(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")

	var id int
	err := h.db.QueryRow("SELECT id FROM players WHERE username = $1", username).Scan(&id)
	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	} else if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	actorURI := fmt.Sprintf("%s/users/%s", h.cfg.BaseURL, username)
	actor := bik.Actor{
		Context:           bik.APContext,
		Type:              "Person",
		ID:                actorURI,
		PreferredUsername: username,
		Inbox:             fmt.Sprintf("%s/bik/inbox", h.cfg.BaseURL),
		Followers:         fmt.Sprintf("%s/users/%s/followers", h.cfg.BaseURL, username),
	}

	w.Header().Set("Content-Type", "application/activity+json")
	if err := json.NewEncoder(w).Encode(actor); err != nil {
		log.Printf("ActorDocument: encode: %v", err)
	}
}

// WellKnownBIKKeys handles GET /.well-known/bik/keys
func (h *Handler) WellKnownBIKKeys(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(h.bikKeys.JWKSet()); err != nil {
		log.Printf("WellKnownBIKKeys: encode: %v", err)
	}
}

// BIKInbox handles POST /bik/inbox — receives AP activities from remote servers
func (h *Handler) BIKInbox(w http.ResponseWriter, r *http.Request) {
	var activity bik.Activity
	if err := json.NewDecoder(r.Body).Decode(&activity); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Verify actor is from a known peer to prevent forged activities
	if activity.Actor != "" {
		actorURL, err := url.Parse(activity.Actor)
		if err != nil || actorURL.Host == "" {
			http.Error(w, "invalid actor URI", http.StatusBadRequest)
			return
		}
		isPeer := false
		for _, peer := range h.cfg.BIKPeers {
			peerURL, err := url.Parse(peer)
			if err != nil {
				continue
			}
			if peerURL.Scheme == actorURL.Scheme && peerURL.Host == actorURL.Host {
				isPeer = true
				break
			}
		}
		if !isPeer {
			log.Printf("BIKInbox: rejected activity from unknown peer %s://%s", actorURL.Scheme, actorURL.Host)
			http.Error(w, "actor not from a known peer", http.StatusForbidden)
			return
		}
	}

	switch activity.Type {
	case "Accept":
		h.handleChallengeAccept(w, r, activity)
	case "Create":
		h.handleRemoteCreate(w, r, activity)
	case "Undo":
		h.handleRemoteUndo(w, r, activity)
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
		challengeID     int
		creatorID       int
		boardSize       int
		timeCtrlRaw     []byte
		colorAssignment string
		status          string
		expiresAt       time.Time
	)
	err := h.db.QueryRow(`
		SELECT id, creator_id, board_size, time_control, color_assignment, status, expires_at
		FROM bik_challenges WHERE uri = $1
	`, challengeURI).Scan(&challengeID, &creatorID, &boardSize, &timeCtrlRaw, &colorAssignment, &status, &expiresAt)
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
	if err := h.db.QueryRow("SELECT username FROM players WHERE id = $1", creatorID).Scan(&creatorUsername); err != nil {
		http.Error(w, "failed to look up creator", http.StatusInternalServerError)
		return
	}

	// Assign colors based on the challenge's colorAssignment field
	creatorActor := fmt.Sprintf("%s/users/%s", h.cfg.BaseURL, creatorUsername)
	acceptorActor := activity.Actor
	var blackActor, whiteActor string
	switch colorAssignment {
	case "white":
		// Creator requested white
		blackActor = acceptorActor
		whiteActor = creatorActor
	case "random":
		if mathrand.Intn(2) == 0 {
			blackActor = creatorActor
			whiteActor = acceptorActor
		} else {
			blackActor = acceptorActor
			whiteActor = creatorActor
		}
	default: // "black" or unset
		blackActor = creatorActor
		whiteActor = acceptorActor
	}

	// Create the game — set local player_id on the correct color column
	var dbGameID int
	creatorIsBlack := blackActor == creatorActor
	if creatorIsBlack {
		err = h.db.QueryRow(`
			INSERT INTO games (black_player_id, board_size, status, creator_id, black_actor_uri, white_actor_uri, bik_challenge_uri)
			VALUES ($1, $2, 'active', $1, $3, $4, $5)
			RETURNING id
		`, creatorID, boardSize, blackActor, whiteActor, challengeURI).Scan(&dbGameID)
	} else {
		err = h.db.QueryRow(`
			INSERT INTO games (white_player_id, board_size, status, creator_id, black_actor_uri, white_actor_uri, bik_challenge_uri)
			VALUES ($1, $2, 'active', $1, $3, $4, $5)
			RETURNING id
		`, creatorID, boardSize, blackActor, whiteActor, challengeURI).Scan(&dbGameID)
	}
	if err != nil {
		http.Error(w, "failed to create game", http.StatusInternalServerError)
		return
	}
	gameID := fmt.Sprintf("%d", dbGameID)

	// Mark challenge as accepted
	if _, err := h.db.Exec("UPDATE bik_challenges SET status = 'accepted' WHERE id = $1", challengeID); err != nil {
		log.Printf("handleChallengeAccept: mark accepted: %v", err)
	}

	// Build response: CreateGame activity
	gameURI := fmt.Sprintf("%s/games/%s", h.cfg.BaseURL, gameID)
	baseURL, _ := url.Parse(h.cfg.BaseURL)
	wsScheme := "ws"
	if baseURL.Scheme == "https" {
		wsScheme = "wss"
	}
	wsURL := fmt.Sprintf("%s://%s/ws/games/%s", wsScheme, baseURL.Host, gameID)
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
				ID:        gameURI,
				Black:     blackActor,
				White:     whiteActor,
				WebSocket: wsURL,
			},
		},
	}

	// Also broadcast Undo for the challenge so followers know it's taken
	go h.broadcastUndoChallenge(challengeURI, blackActor)

	w.Header().Set("Content-Type", "application/activity+json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(gameActivity); err != nil {
		log.Printf("handleChallengeAccept: encode: %v", err)
	}
}

// handleRemoteCreate processes a Create activity from a peer — stores a remote challenge.
func (h *Handler) handleRemoteCreate(w http.ResponseWriter, r *http.Request, activity bik.Activity) {
	// Re-marshal and parse the object as a Note with BikChallenge attachment
	objBytes, err := json.Marshal(activity.Object)
	if err != nil {
		http.Error(w, "invalid object", http.StatusBadRequest)
		return
	}

	var note struct {
		Type       string `json:"type"`
		ID         string `json:"id"`
		Attachment struct {
			Type            string          `json:"type"`
			BoardSize       int             `json:"boardSize"`
			TimeControl     json.RawMessage `json:"timeControl"`
			ColorAssignment string          `json:"colorAssignment"`
			ExpiresAt       time.Time       `json:"expiresAt"`
		} `json:"attachment"`
	}
	if err := json.Unmarshal(objBytes, &note); err != nil || note.Type != "Note" {
		http.Error(w, "expected Note object", http.StatusBadRequest)
		return
	}
	if note.Attachment.Type != "BikChallenge" {
		http.Error(w, "expected BikChallenge attachment", http.StatusBadRequest)
		return
	}

	// Discover the actor's inbox from their actor document
	inboxURL := h.discoverInbox(activity.Actor)
	if inboxURL == "" {
		// Fall back: assume standard path
		idx := strings.LastIndex(activity.Actor, "/users/")
		if idx < 0 {
			http.Error(w, "could not determine inbox URL for actor", http.StatusBadRequest)
			return
		}
		inboxURL = activity.Actor[:idx] + "/bik/inbox"
	}

	_, err = h.db.Exec(`
		INSERT INTO remote_challenges (uri, actor_uri, board_size, time_control, color_assignment, inbox_url, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (uri) DO NOTHING
	`, note.ID, activity.Actor, note.Attachment.BoardSize, string(note.Attachment.TimeControl),
		note.Attachment.ColorAssignment, inboxURL, note.Attachment.ExpiresAt)
	if err != nil {
		log.Printf("handleRemoteCreate: insert: %v", err)
		http.Error(w, "failed to store challenge", http.StatusInternalServerError)
		return
	}

	log.Printf("federation: stored remote challenge %s from %s", note.ID, activity.Actor)
	w.WriteHeader(http.StatusAccepted)
}

// handleRemoteUndo processes an Undo activity — removes a remote challenge.
func (h *Handler) handleRemoteUndo(w http.ResponseWriter, r *http.Request, activity bik.Activity) {
	challengeURI, ok := activity.Object.(string)
	if !ok {
		http.Error(w, "object must be a URI string", http.StatusBadRequest)
		return
	}

	result, err := h.db.Exec("DELETE FROM remote_challenges WHERE uri = $1", challengeURI)
	if err != nil {
		log.Printf("handleRemoteUndo: delete: %v", err)
		http.Error(w, "failed to remove challenge", http.StatusInternalServerError)
		return
	}
	rows, _ := result.RowsAffected()
	log.Printf("federation: undo challenge %s (deleted %d rows)", challengeURI, rows)
	w.WriteHeader(http.StatusAccepted)
}

// discoverInbox fetches an actor document and returns the inbox URL.
func (h *Handler) discoverInbox(actorURI string) string {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", actorURI, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Accept", "application/activity+json")
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	var actor bik.Actor
	if err := json.NewDecoder(resp.Body).Decode(&actor); err != nil {
		return ""
	}
	return actor.Inbox
}

func (h *Handler) broadcastUndoChallenge(challengeURI, actor string) {
	activity := bik.Activity{
		Context: bik.APContext,
		Type:    "Undo",
		Actor:   actor,
		Object:  challengeURI,
	}
	h.deliverToPeers(activity)
}

// CreateBIKChallenge handles POST /api/v1/bik/challenges
func (h *Handler) CreateBIKChallenge(w http.ResponseWriter, r *http.Request) {
	playerID, _ := middleware.GetPlayerID(r)

	var req struct {
		BoardSize       int             `json:"boardSize"`
		TimeControl     bik.TimeControl `json:"timeControl"`
		ColorAssignment string          `json:"colorAssignment"`
		ExpiresIn       int             `json:"expiresIn"` // seconds
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
	if req.ColorAssignment == "" {
		req.ColorAssignment = "random"
	}

	var username string
	if err := h.db.QueryRow("SELECT username FROM players WHERE id = $1", playerID).Scan(&username); err != nil {
		http.Error(w, "failed to look up player", http.StatusInternalServerError)
		return
	}

	var idBytes [16]byte
	if _, err := rand.Read(idBytes[:]); err != nil {
		http.Error(w, "failed to generate ID", http.StatusInternalServerError)
		return
	}
	challengeID := hex.EncodeToString(idBytes[:])
	challengeURI := fmt.Sprintf("%s/challenges/%s", h.cfg.BaseURL, challengeID)
	expiresAt := time.Now().Add(time.Duration(req.ExpiresIn) * time.Second)

	timeCtrlJSON, _ := json.Marshal(req.TimeControl)
	_, err := h.db.Exec(`
		INSERT INTO bik_challenges (creator_id, uri, board_size, time_control, color_assignment, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, playerID, challengeURI, req.BoardSize, timeCtrlJSON, req.ColorAssignment, expiresAt)
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
				Type:            "BikChallenge",
				BoardSize:       req.BoardSize,
				TimeControl:     req.TimeControl,
				ColorAssignment: req.ColorAssignment,
				ExpiresAt:       expiresAt,
			},
		},
	}

	// Push to peer servers
	h.deliverToPeers(activity)

	w.Header().Set("Content-Type", "application/activity+json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(activity); err != nil {
		log.Printf("CreateBIKChallenge: encode: %v", err)
	}
}

// ListBIKChallenges handles GET /api/v1/bik/challenges
func (h *Handler) ListBIKChallenges(w http.ResponseWriter, r *http.Request) {
	type challengeItem struct {
		URI             string          `json:"uri"`
		Creator         string          `json:"creator"`
		BoardSize       int             `json:"boardSize"`
		TimeCtrl        json.RawMessage `json:"timeControl"`
		ColorAssignment string          `json:"colorAssignment"`
		ExpiresAt       time.Time       `json:"expiresAt"`
		Remote          bool            `json:"remote"`
		InboxURL        string          `json:"inboxUrl,omitempty"`
	}

	items := []challengeItem{}

	// Local challenges
	rows, err := h.db.Query(`
		SELECT c.uri, p.username, c.board_size, c.time_control, c.color_assignment, c.expires_at
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

	for rows.Next() {
		var item challengeItem
		var timeCtrlRaw []byte
		if err := rows.Scan(&item.URI, &item.Creator, &item.BoardSize, &timeCtrlRaw, &item.ColorAssignment, &item.ExpiresAt); err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		item.TimeCtrl = json.RawMessage(timeCtrlRaw)
		items = append(items, item)
	}

	// Remote challenges from peers
	remoteRows, err := h.db.Query(`
		SELECT uri, actor_uri, board_size, time_control, color_assignment, inbox_url, expires_at
		FROM remote_challenges
		WHERE expires_at > NOW()
		ORDER BY received_at DESC
	`)
	if err != nil {
		log.Printf("ListBIKChallenges: remote query: %v", err)
	} else {
		defer remoteRows.Close()
		for remoteRows.Next() {
			var item challengeItem
			var timeCtrlRaw []byte
			if err := remoteRows.Scan(&item.URI, &item.Creator, &item.BoardSize, &timeCtrlRaw, &item.ColorAssignment, &item.InboxURL, &item.ExpiresAt); err != nil {
				log.Printf("ListBIKChallenges: scan remote: %v", err)
				continue
			}
			item.TimeCtrl = json.RawMessage(timeCtrlRaw)
			item.Remote = true
			items = append(items, item)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(items); err != nil {
		log.Printf("ListBIKChallenges: encode: %v", err)
	}
}

// GetBIKToken handles POST /api/v1/bik/token — issues a signed token for a remote game
func (h *Handler) GetBIKToken(w http.ResponseWriter, r *http.Request) {
	playerID, _ := middleware.GetPlayerID(r)
	gameID := chi.URLParam(r, "gameID")

	var username string
	if err := h.db.QueryRow("SELECT username FROM players WHERE id = $1", playerID).Scan(&username); err != nil {
		http.Error(w, "failed to look up player", http.StatusInternalServerError)
		return
	}

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
	if err := json.NewEncoder(w).Encode(map[string]string{"token": token}); err != nil {
		log.Printf("GetBIKToken: encode: %v", err)
	}
}

// AcceptRemoteChallenge handles POST /api/v1/bik/challenges/accept
// The local server proxies the Accept to the remote server's inbox on behalf of the user,
// creates a local game record, and returns the local game ID.
func (h *Handler) AcceptRemoteChallenge(w http.ResponseWriter, r *http.Request) {
	playerID, _ := middleware.GetPlayerID(r)

	var req struct {
		ChallengeURI string `json:"challengeUri"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Look up the trusted inbox URL and board size from the stored remote challenge.
	// Never trust a client-supplied inbox URL — that's an SSRF vector.
	var inboxURL string
	var boardSize int
	if err := h.db.QueryRow(
		"SELECT inbox_url, board_size FROM remote_challenges WHERE uri = $1",
		req.ChallengeURI,
	).Scan(&inboxURL, &boardSize); err != nil {
		http.Error(w, "challenge not found", http.StatusNotFound)
		return
	}

	var username string
	if err := h.db.QueryRow("SELECT username FROM players WHERE id = $1", playerID).Scan(&username); err != nil {
		http.Error(w, "failed to look up player", http.StatusInternalServerError)
		return
	}

	actorURI := fmt.Sprintf("%s/users/%s", h.cfg.BaseURL, username)
	acceptActivity := bik.Activity{
		Context: bik.APContext,
		Type:    "Accept",
		Actor:   actorURI,
		Object:  req.ChallengeURI,
	}

	body, err := json.Marshal(acceptActivity)
	if err != nil {
		http.Error(w, "failed to marshal activity", http.StatusInternalServerError)
		return
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Post(inboxURL, "application/activity+json", bytes.NewReader(body))
	if err != nil {
		log.Printf("AcceptRemoteChallenge: POST to %s: %v", inboxURL, err)
		http.Error(w, fmt.Sprintf("failed to reach remote server: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		http.Error(w, fmt.Sprintf("remote server returned %d: %s", resp.StatusCode, string(respBody)), resp.StatusCode)
		return
	}

	// Parse the Create game activity from the remote server
	var gameActivity struct {
		Object struct {
			Attachment struct {
				Type      string `json:"type"`
				ID        string `json:"id"`
				Black     string `json:"black"`
				White     string `json:"white"`
				WebSocket string `json:"websocket"`
			} `json:"attachment"`
		} `json:"object"`
	}
	if err := json.Unmarshal(respBody, &gameActivity); err != nil {
		log.Printf("AcceptRemoteChallenge: parse response: %v", err)
		http.Error(w, "failed to parse game response", http.StatusBadGateway)
		return
	}

	att := gameActivity.Object.Attachment
	if att.Type != "BikGame" {
		http.Error(w, "unexpected response from remote server", http.StatusBadGateway)
		return
	}

	// Verify the local actor is one of the two listed players
	if att.Black != actorURI && att.White != actorURI {
		http.Error(w, "remote server returned a game we are not part of", http.StatusBadGateway)
		return
	}

	// Verify the WebSocket URL has a safe scheme to avoid open-redirect / XSS
	wsURL, err := url.Parse(att.WebSocket)
	if err != nil || (wsURL.Scheme != "ws" && wsURL.Scheme != "wss") {
		http.Error(w, "remote server returned an invalid WebSocket URL", http.StatusBadGateway)
		return
	}

	// Determine which color the local player is
	var localGameID int
	if att.Black == actorURI {
		err = h.db.QueryRow(`
			INSERT INTO games (black_player_id, board_size, status, black_actor_uri, white_actor_uri, bik_challenge_uri, remote_game_uri, remote_ws_url)
			VALUES ($1, $2, 'active', $3, $4, $5, $6, $7)
			RETURNING id
		`, playerID, boardSize, att.Black, att.White, req.ChallengeURI, att.ID, att.WebSocket).Scan(&localGameID)
	} else {
		err = h.db.QueryRow(`
			INSERT INTO games (white_player_id, board_size, status, black_actor_uri, white_actor_uri, bik_challenge_uri, remote_game_uri, remote_ws_url)
			VALUES ($1, $2, 'active', $3, $4, $5, $6, $7)
			RETURNING id
		`, playerID, boardSize, att.Black, att.White, req.ChallengeURI, att.ID, att.WebSocket).Scan(&localGameID)
	}
	if err != nil {
		log.Printf("AcceptRemoteChallenge: create local game: %v", err)
		http.Error(w, "failed to create local game record", http.StatusInternalServerError)
		return
	}

	// Remove the remote challenge from our list
	if _, err := h.db.Exec("DELETE FROM remote_challenges WHERE uri = $1", req.ChallengeURI); err != nil {
		log.Printf("AcceptRemoteChallenge: delete remote challenge: %v", err)
	}

	log.Printf("federation: accepted challenge %s → local game %d (remote %s)", req.ChallengeURI, localGameID, att.ID)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"gameId":      localGameID,
		"remoteWsUrl": att.WebSocket,
	}); err != nil {
		log.Printf("AcceptRemoteChallenge: encode: %v", err)
	}
}
