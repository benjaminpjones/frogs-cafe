package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"frogs_cafe/auth"

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
	playerID int
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
	// Get token from query parameter or header
	token := r.URL.Query().Get("token")
	if token == "" {
		authHeader := r.Header.Get("Authorization")
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token = authHeader[7:]
		}
	}

	// Token is now optional - guests can view without authentication
	var playerID int
	var username string
	var err error

	if token != "" {
		// Validate session token
		playerID, username, err = auth.ValidateSession(h.db.DB, token)
		if err != nil {
			log.Printf("Token validation failed: %v", err)
			// Continue as guest instead of returning error
			playerID = 0
			username = "guest"
		}
	} else {
		// Guest viewer
		playerID = 0
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
		gameID:   r.URL.Query().Get("game_id"),
		userID:   username,
		playerID: playerID,
	}

	hub.register <- client

	go client.writePump()
	go client.readPump()
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
			// Only authenticated players can make moves
			if c.playerID == 0 {
				log.Printf("Guest attempted to make a move - rejected")
				continue
			}

			if data, ok := msg["data"].(map[string]interface{}); ok {
				// Use authenticated playerID from JWT, not from message
				if err := hub.handler.SaveMove(c.gameID, c.playerID, data); err != nil {
					log.Printf("Error saving move: %v", err)
					continue
				}

				// Add the authenticated player_id to the data before broadcasting
				data["player_id"] = float64(c.playerID) // JSON numbers are float64
				msg["data"] = data

				// Re-marshal the updated message
				updatedMessage, err := json.Marshal(msg)
				if err != nil {
					log.Printf("Error marshaling updated message: %v", err)
					continue
				}

				// Broadcast the updated message to all clients in the same game
				hub.broadcast <- updatedMessage
				continue
			}
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
