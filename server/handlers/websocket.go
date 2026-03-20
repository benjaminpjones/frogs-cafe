package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"frogs_cafe/auth"
	"frogs_cafe/bik"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins in development - restrict in production
		return true
	},
}

type Client struct {
	conn     *websocket.Conn
	send     chan []byte
	gameID   string
	userID   string
	playerID int    // local player ID; -1 for remote (BIK) players
	actorURI string // set for remote players authenticated via BIK token
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mutex      sync.RWMutex
	handler    *Handler
}

var hub *Hub

func InitHub(h *Handler) {
	hub = &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		handler:    h,
	}
	go hub.run()
}

func GetHub() *Hub {
	return hub
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client] = true
			h.mutex.Unlock()
			log.Printf("Client registered: %s", client.userID)

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				log.Printf("Client unregistered: %s", client.userID)
			}
			h.mutex.Unlock()

		case message := <-h.broadcast:
			h.mutex.RLock()
			// Parse message to log type and determine target game
			var msgData map[string]interface{}
			targetGameID := ""
			msgType := "unknown"
			if err := json.Unmarshal(message, &msgData); err == nil {
				if t, ok := msgData["type"].(string); ok {
					msgType = t
				}
				if data, ok := msgData["data"].(map[string]interface{}); ok {
					if gid, ok := data["game_id"].(float64); ok {
						targetGameID = fmt.Sprintf("%.0f", gid)
					}
				}
			}

			recipientCount := 0
			for client := range h.clients {
				// Only send to clients watching the target game (or broadcast if no game_id)
				if targetGameID == "" || client.gameID == targetGameID {
					select {
					case client.send <- message:
						recipientCount++
					default:
						close(client.send)
						delete(h.clients, client)
					}
				}
			}
			log.Printf("Broadcast [%s] to %d clients (game=%s)", msgType, recipientCount, targetGameID)
			h.mutex.RUnlock()
		}
	}
}

func (h *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		authHeader := r.Header.Get("Authorization")
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token = authHeader[7:]
		}
	}

	// Game ID from URL path (/ws/games/{gameID}) or query param (legacy)
	gameID := chi.URLParam(r, "gameID")
	if gameID == "" {
		gameID = r.URL.Query().Get("game_id")
	}

	var playerID int
	var username string
	var actorURI string

	if token != "" {
		// Try local session token first
		var err error
		playerID, username, err = auth.ValidateSession(h.db.DB, token)
		if err != nil {
			// Try BIK token (remote player)
			bikTok, bikErr := h.verifyBIKToken(token, gameID)
			if bikErr != nil {
				log.Printf("WS auth failed — session: %v; bik: %v", err, bikErr)
				// Fall through as guest
			} else {
				actorURI = bikTok.Player
				username = bikTok.Player
				playerID = -1 // sentinel: remote player
				log.Printf("Remote player authenticated via BIK token: %s", actorURI)
			}
		}
	}

	if playerID == 0 && actorURI == "" {
		username = "guest"
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &Client{
		conn:     conn,
		send:     make(chan []byte, 256),
		gameID:   gameID,
		userID:   username,
		playerID: playerID,
		actorURI: actorURI,
	}

	hub.register <- client

	// Send current game state immediately on connect
	if gameID != "" {
		go h.sendGameState(client, gameID)
	}

	go client.writePump()
	go client.readPump()
}

func (h *Handler) sendGameState(client *Client, gameIDStr string) {
	// Fetch moves
	rows, err := h.db.Query(
		"SELECT x, y FROM moves WHERE game_id = $1 ORDER BY move_number ASC",
		gameIDStr,
	)
	if err != nil {
		log.Printf("sendGameState: query moves: %v", err)
		return
	}
	defer rows.Close()

	type pos [2]int
	moves := []interface{}{}
	for rows.Next() {
		var x, y int
		if err := rows.Scan(&x, &y); err != nil {
			log.Printf("sendGameState: scan move: %v", err)
			continue
		}
		moves = append(moves, pos{x, y})
	}

	// Fetch game phase
	var status string
	h.db.QueryRow("SELECT status FROM games WHERE id = $1", gameIDStr).Scan(&status)
	phase := status // active/finished map directly; waiting → active for simplicity

	nextToPlay := "black"
	if len(moves)%2 == 1 {
		nextToPlay = "white"
	}

	msg := map[string]interface{}{
		"type": "game_state",
		"data": map[string]interface{}{
			"moves":      moves,
			"phase":      phase,
			"nextToPlay": nextToPlay,
			"clock": map[string]interface{}{
				"system":      "absolute",
				"black":       300,
				"white":       300,
				"activeColor": nextToPlay,
			},
		},
	}
	if b, err := json.Marshal(msg); err == nil {
		client.send <- b
	}
}

// verifyBIKToken parses and verifies a BIK token issued by a remote server.
func (h *Handler) verifyBIKToken(token, gameID string) (*bik.BikToken, error) {
	// Decode payload without verifying (to extract the player's server domain)
	dotIdx := strings.LastIndex(token, ".")
	if dotIdx < 0 {
		return nil, fmt.Errorf("malformed BIK token")
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(token[:dotIdx])
	if err != nil {
		return nil, fmt.Errorf("decode BIK token payload: %w", err)
	}
	var t bik.BikToken
	if err := json.Unmarshal(payloadBytes, &t); err != nil {
		return nil, fmt.Errorf("unmarshal BIK token: %w", err)
	}

	// Extract server domain from actor URI (e.g. "https://server-b.example/users/bob" → "server-b.example")
	playerURL, err := url.Parse(t.Player)
	if err != nil || playerURL.Host == "" {
		return nil, fmt.Errorf("invalid player URI in BIK token: %s", t.Player)
	}
	domain := playerURL.Host

	// Fetch remote server's public key (cached)
	pubKey, err := h.keyCache.FetchPublicKey(domain)
	if err != nil {
		return nil, fmt.Errorf("fetch public key for %s: %w", domain, err)
	}

	// Verify signature and expiry
	verified, err := bik.VerifyToken(token, pubKey)
	if err != nil {
		return nil, err
	}

	// Confirm the token is for this game
	expectedGameURI := fmt.Sprintf("%s/games/%s", h.cfg.BaseURL, gameID)
	if verified.Game != expectedGameURI {
		return nil, fmt.Errorf("BIK token game mismatch: got %s, want %s", verified.Game, expectedGameURI)
	}

	return verified, nil
}

func (c *Client) readPump() {
	defer func() {
		hub.unregister <- c
		if err := c.conn.Close(); err != nil {
			log.Printf("Failed to close connection: %v", err)
		}
	}()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Process message (game move, chat, etc.)
		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("JSON unmarshal error: %v", err)
			continue
		}

		log.Printf("Received message: %+v", msg)

		// Handle authentication upgrade
		if msgType, ok := msg["type"].(string); ok && msgType == "authenticate" {
			if data, ok := msg["data"].(map[string]interface{}); ok {
				if tokenStr, ok := data["token"].(string); ok {
					// Validate the token
					playerID, username, err := auth.ValidateSession(hub.handler.db.DB, tokenStr)
					if err != nil {
						log.Printf("Authentication upgrade failed: %v", err)
						// Send error response back to client
						response := map[string]interface{}{
							"type": "auth_error",
							"data": map[string]string{"error": "Invalid token"},
						}
						if respBytes, err := json.Marshal(response); err == nil {
							c.send <- respBytes
						}
						continue
					}

					// Upgrade the client's credentials
					c.playerID = playerID
					c.userID = username
					log.Printf("Client upgraded to authenticated user: %s (ID: %d)", username, playerID)

					// Send success response
					response := map[string]interface{}{
						"type": "auth_success",
						"data": map[string]interface{}{
							"player_id": playerID,
							"username":  username,
						},
					}
					if respBytes, err := json.Marshal(response); err == nil {
						c.send <- respBytes
					}
					continue
				}
			}
		}

		// Handle move type messages
		if msgType, ok := msg["type"].(string); ok && msgType == "move" {
			// Guests cannot make moves
			if c.playerID == 0 && c.actorURI == "" {
				log.Printf("Guest attempted to make a move - rejected")
				continue
			}

			// Verify the client is actually a participant in this game
			ok, err := hub.handler.isGameParticipant(c.gameID, c.playerID, c.actorURI)
			if err != nil || !ok {
				log.Printf("Move rejected: %s is not a participant in game %s", c.userID, c.gameID)
				reject := map[string]interface{}{
					"type": "move_rejected",
					"data": map[string]string{"reason": "not_your_turn"},
				}
				if b, err := json.Marshal(reject); err == nil {
					c.send <- b
				}
				continue
			}

			if data, ok := msg["data"].(map[string]interface{}); ok {
				moveNumber, err := hub.handler.SaveMove(c.gameID, c.playerID, c.actorURI, data)
				if err != nil {
					log.Printf("Error saving move: %v", err)
					continue
				}

				// Determine color from move number: odd = black, even = white
				color := "black"
				nextToPlay := "white"
				if moveNumber%2 == 0 {
					color = "white"
					nextToPlay = "black"
				}

				// Build move_played broadcast per BIK spec
				movePlayed := map[string]interface{}{
					"type": "move_played",
					"data": map[string]interface{}{
						"pos":        data["pos"],
						"moveNumber": moveNumber,
						"color":      color,
						"captures":   []interface{}{},
						"nextToPlay": nextToPlay,
						"clock": map[string]interface{}{
							"system":      "absolute",
							"black":       300,
							"white":       300,
							"activeColor": nextToPlay,
						},
					},
				}

				updatedMessage, err := json.Marshal(movePlayed)
				if err != nil {
					log.Printf("Error marshaling move_played: %v", err)
					continue
				}

				hub.broadcast <- updatedMessage
				continue
			}
		}

		// Handle resign
		if msgType, ok := msg["type"].(string); ok && msgType == "resign" {
			if c.playerID == 0 && c.actorURI == "" {
				continue // guests can't resign
			}
			ok, _ := hub.handler.isGameParticipant(c.gameID, c.playerID, c.actorURI)
			if !ok {
				continue
			}
			hub.handler.handleResign(c)
			continue
		}

		// For other message types, broadcast as-is
		hub.broadcast <- message
	}
}

func (c *Client) writePump() {
	defer func() {
		if err := c.conn.Close(); err != nil {
			log.Printf("Failed to close connection: %v", err)
		}
	}()

	for message := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Printf("Write error: %v", err)
			return
		}
	}
}
