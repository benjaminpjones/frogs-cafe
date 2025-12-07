package handlers

import (
	"encoding/json"
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
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
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

	if token == "" {
		http.Error(w, "Unauthorized: no token provided", http.StatusUnauthorized)
		return
	}

	// Validate session token
	playerID, username, err := auth.ValidateSession(h.db.DB, token)
	if err != nil {
		http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
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
		c.conn.Close()
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

		// Handle move type messages
		if msgType, ok := msg["type"].(string); ok && msgType == "move" {
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
		c.conn.Close()
	}()

	for message := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Printf("Write error: %v", err)
			return
		}
	}
}
