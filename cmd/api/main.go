package main

import (
	"log"
	"net/http"
	"os"

	"gwens-bridal-api/internal/handlers"
	"gwens-bridal-api/internal/middleware"
	"gwens-bridal-api/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware/cors"
	"github.com/joho/godotenv"
)

func main() {
	
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	db, err := storage.NewPostgresDB()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	s3Client, err := storage.NewS3Client()
	if err != nil {
		log.Fatal("Failed to initialize S3 client:", err)
	}

	authHandler := handlers.NewAuthHandler()
	imageHandler := handlers.NewImageHandler(db, s3Client)
	
	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"}, 
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	
	r.Post("/api/auth/login", authHandler.Login)
	r.Get("/api/images", imageHandler.GetImages) 

	r.Route("/api/admin", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware)
		r.Post("/images", imageHandler.UploadImage)
		r.Delete("/images/{id}", imageHandler.DeleteImage)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}