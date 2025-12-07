package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"frogs_cafe/auth"
	"frogs_cafe/config"
	"frogs_cafe/database"
	"frogs_cafe/handlers"
	"frogs_cafe/middleware"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Load configuration
	cfg := config.Load()

	// Initialize database
	db, err := database.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := database.RunMigrations(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Start session cleanup goroutine (runs every hour)
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		
		for range ticker.C {
			if err := auth.CleanupExpiredSessions(db.DB); err != nil {
				log.Printf("Failed to cleanup expired sessions: %v", err)
			} else {
				log.Println("Cleaned up expired sessions")
			}
		}
	}()

	// Initialize router
	r := chi.NewRouter()

	// Middleware
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.RequestID)

	// CORS configuration for development
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Initialize handlers
	h := handlers.New(db)

	// Routes
	r.Get("/health", h.HealthCheck)
	
	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Auth routes
		r.Post("/register", h.Register)
		r.Post("/login", h.Login)
		r.Post("/logout", h.Logout)
		
		// Public game routes (no auth required)
		r.Get("/games", h.ListGames)
		r.Get("/games/{gameID}", h.GetGame)
		r.Get("/games/{gameID}/moves", h.GetGameMoves)
		
		// Protected game routes (require authentication)
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth(db.DB))
			r.Post("/games", h.CreateGame)
			r.Post("/games/{gameID}/join", h.JoinGame)
		})
		
		// Player routes
		r.Get("/players", h.ListPlayers)
		r.Post("/players", h.CreatePlayer)
		r.Get("/players/{playerID}", h.GetPlayer)
	})

	// WebSocket route
	r.Get("/ws", h.HandleWebSocket)

	// Serve static files from React build (production)
	staticDir := "./static"
	if _, err := os.Stat(staticDir); err == nil {
		fileServer := http.FileServer(http.Dir(staticDir))
		r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
			// Check if file exists
			filePath := staticDir + r.URL.Path
			if _, err := os.Stat(filePath); os.IsNotExist(err) || r.URL.Path == "/" {
				// Serve index.html for SPA routing
				http.ServeFile(w, r, staticDir+"/index.html")
				return
			}
			fileServer.ServeHTTP(w, r)
		})
		log.Println("Serving static files from", staticDir)
	}

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting server on port %s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
