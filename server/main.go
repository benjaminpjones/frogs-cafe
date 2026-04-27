package main

import (
	"log"
	"net/http"
	"os"
	"strings"
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
	log.Println("\n=== Starting Frogs Café Server ===")
	log.Printf("Current working directory: %s", os.Getenv("PWD"))
	log.Println("Attempting to load .env file...")

	if err := godotenv.Load(); err != nil {
		log.Println("[INFO] No .env file found, using system environment variables")
	} else {
		log.Println("[OK] .env file loaded successfully")
	}

	// Load configuration
	log.Println("\nLoading configuration...")
	cfg := config.Load()

	// Initialize database
	db, err := database.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("Failed to close database: %v", err)
		}
	}()

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

	// CORS configuration — only allow any localhost origin in development
	isDev := cfg.Environment == "development"
	r.Use(cors.Handler(cors.Options{
		AllowOriginFunc: func(r *http.Request, origin string) bool {
			return isDev && strings.HasPrefix(origin, "http://localhost:")
		},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Initialize handlers
	h, err := handlers.New(db, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize handlers: %v", err)
	}

	// Routes
	r.Get("/health", h.HealthCheck)

	// Federation endpoints
	r.Get("/.well-known/webfinger", h.WellKnownWebFinger)
	r.Get("/.well-known/bik/keys", h.WellKnownBIKKeys)
	r.Post("/bik/inbox", h.BIKInbox)
	r.Get("/users/{username}", h.ActorDocument)

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

		// BIK routes
		r.Get("/bik/challenges", h.ListBIKChallenges)
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth(db.DB))
			r.Post("/bik/challenges", h.CreateBIKChallenge)
			r.Post("/bik/token/{gameID}", h.GetBIKToken)
		})
	})

	// WebSocket route
	r.Get("/ws", h.HandleWebSocket)
	r.Get("/ws/games/{gameID}", h.HandleWebSocket)

	// Serve static files from React build (production)
	staticDir := "./static"
	if _, err := os.Stat(staticDir); err == nil {
		fileServer := http.FileServer(http.Dir(staticDir))
		r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
			// Check if file exists
			filePath := staticDir + r.URL.Path
			if _, err := os.Stat(filePath); os.IsNotExist(err) || r.URL.Path == "/" {
				// Serve index.html for SPA routing - no cache for HTML
				w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
				w.Header().Set("Pragma", "no-cache")
				w.Header().Set("Expires", "0")
				http.ServeFile(w, r, staticDir+"/index.html")
				return
			}
			// Cache static assets (JS, CSS, images) for 1 year since they have hashed names
			if strings.HasPrefix(r.URL.Path, "/assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
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
