package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"

	"ticket-system/internal/handlers"
	"ticket-system/internal/middleware"
	"ticket-system/internal/store"
)

func main() {
	// Load environment variables from .env file if it exists
	loadEnv()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "default_super_secret_jwt_key"
		log.Println("WARNING: JWT_SECRET environment variable is not set. Using default secret key.")
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "tickets.db"
	}

	// Initialize the Store (SQLite database)
	log.Printf("Connecting to database at: %s", dbPath)
	dbStore, err := store.NewStore(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize store: %v", err)
	}
	defer dbStore.Close()

	// Initialize Handlers
	h := handlers.NewHandlers(dbStore, jwtSecret)

	// Set up Chi router
	r := chi.NewRouter()

	// Standard middlewares
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(corsMiddleware) // Enable Cross-Origin Resource Sharing globally

	// Public Routes
	r.Get("/health", h.Health)
	r.Post("/auth/register", h.Register)
	r.Post("/auth/login", h.Login)

	// Protected Routes
	r.Route("/tickets", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(jwtSecret))
		r.Post("/", h.CreateTicket)
		r.Get("/", h.ListTickets)
		r.Get("/{id}", h.GetTicketByID)
		r.Patch("/{id}/status", h.UpdateTicketStatus)
	})

	// Serve static files from React build directory in production
	distPath := filepath.Join(".", "frontend", "dist")
	if _, err := os.Stat(distPath); err == nil {
		log.Printf("Serving frontend static assets from %s", distPath)
		fileServer(r, "/", http.Dir(distPath))
	} else {
		log.Println("Frontend static asset directory not found; API-only mode active.")
	}

	log.Printf("Server listening on port %s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// corsMiddleware configures CORS headers to allow browser requests from our local React dev environment.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// fileServer sets up a static file server with SPA (Single Page Application) routing fallback.
func fileServer(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("fileServer does not permit any URL parameters.")
	}

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", 301).ServeHTTP)
		path += "/"
	}
	pathPrefix := path
	path += "*"

	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPattern := rctx.RoutePattern()
		trimPrefix := strings.TrimSuffix(pathPattern, "/*")

		// Get path requested relative to the prefix
		reqPath := r.URL.Path
		if strings.HasPrefix(reqPath, trimPrefix) {
			reqPath = reqPath[len(trimPrefix):]
		}
		if reqPath == "" {
			reqPath = "/"
		}

		// Check if file exists in the file system
		f, err := root.Open(reqPath)
		if err != nil {
			// If file does not exist, serve index.html (SPA fallback)
			indexFile, err := root.Open("/index.html")
			if err != nil {
				http.Error(w, "index.html not found", http.StatusNotFound)
				return
			}
			defer indexFile.Close()

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			http.ServeContent(w, r, "index.html", time.Now(), indexFile)
			return
		}
		f.Close()

		// Serve the actual file using http.FileServer
		http.StripPrefix(pathPrefix, http.FileServer(root)).ServeHTTP(w, r)
	})
}

// loadEnv parses a local .env file manually if it exists to keep dependencies minimal
func loadEnv() {
	content, err := os.ReadFile(".env")
	if err != nil {
		// No .env file found; proceed with host environment variables
		return
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		// Strip surrounding quotes if present
		val = strings.Trim(val, `"'`)

		// Only set if not already defined in the system environment
		if os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
}

