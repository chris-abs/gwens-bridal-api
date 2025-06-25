package main

import (
	"log"
	"net/http"
	"os"

	"gwens-bridal-api/internal/handlers"
	"gwens-bridal-api/internal/middleware"
	"gwens-bridal-api/internal/storage"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Initialize database
	db, err := storage.NewPostgresDB()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Initialize S3 client
	s3Client, err := storage.NewS3Client()
	if err != nil {
		log.Fatal("Failed to initialize S3 client:", err)
	}

	// Initialize handlers
	imageHandler := handlers.NewImageHandler(db, s3Client)
	authHandler := handlers.NewAuthHandler()

	// Setup router
	r := chi.NewRouter()

	// Middleware
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.RequestID)

	// CORS for your frontend
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization")

			if r.Method == "OPTIONS" {
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	// Public routes
	r.Get("/images", imageHandler.GetImages)

	// Auth routes
	r.Post("/login", authHandler.Login)

	// Protected admin routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware)
		r.Post("/upload", imageHandler.UploadImage)
		r.Delete("/images/{id}", imageHandler.DeleteImage)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}