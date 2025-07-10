package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gwens-bridal-api/internal/models"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-chi/chi/v5"
)

type ImageHandler struct {
	db       *sql.DB
	s3Client *s3.Client
}

func NewImageHandler(db *sql.DB, s3Client *s3.Client) *ImageHandler {
	return &ImageHandler{
		db:       db,
		s3Client: s3Client,
	}
}

func (h *ImageHandler) GetImages(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")

	var query string
	var args []interface{}

	if category != "" {
		query = `SELECT id, filename, s3_key, s3_url, category, created_at, is_active 
				FROM images 
				WHERE category = $1 AND is_active = true 
				ORDER BY created_at DESC`
		args = []interface{}{category}
	} else {
		query = `SELECT id, filename, s3_key, s3_url, category, created_at, is_active 
				FROM images 
				WHERE is_active = true 
				ORDER BY created_at DESC`
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var images []models.Image
	for rows.Next() {
		var img models.Image
		err := rows.Scan(&img.ID, &img.Filename, &img.S3Key, &img.S3URL, &img.Category, &img.CreatedAt, &img.IsActive)
		if err != nil {
			http.Error(w, "Database scan error", http.StatusInternalServerError)
			return
		}
		images = append(images, img)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(images)
}

func (h *ImageHandler) UploadImage(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20) 
	if err != nil {
		http.Error(w, "File too large", http.StatusBadRequest)
		return
	}

	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "No file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	category := r.FormValue("category")
	if category == "" {
		http.Error(w, "Category is required", http.StatusBadRequest)
		return
	}

	if !isValidImageType(fileHeader.Filename) {
		http.Error(w, "Invalid file type. Only JPG, JPEG, PNG, and WEBP are allowed", http.StatusBadRequest)
		return
	}

	ext := filepath.Ext(fileHeader.Filename)
	filename := fmt.Sprintf("%d_%s%s", time.Now().Unix(), strings.ReplaceAll(category, "/", "-"), ext)
	s3Key := fmt.Sprintf("images/%s", filename)

	fileContent, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	bucketName := os.Getenv("S3_BUCKET_NAME")
	if bucketName == "" {
		http.Error(w, "S3 bucket not configured", http.StatusInternalServerError)
		return
	}

	_, err = h.s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(bucketName),
		Key:         aws.String(s3Key),
		Body:        bytes.NewReader(fileContent),
		ContentType: aws.String(getContentType(ext)),
	})
	if err != nil {
		http.Error(w, "Failed to upload to S3", http.StatusInternalServerError)
		return
	}

	s3URL := fmt.Sprintf("https://%s.s3.amazonaws.com/%s", bucketName, s3Key)

	var imageID int
	err = h.db.QueryRow(`
		INSERT INTO images (filename, s3_key, s3_url, category) 
		VALUES ($1, $2, $3, $4) 
		RETURNING id`,
		filename, s3Key, s3URL, category,
	).Scan(&imageID)
	if err != nil {
		http.Error(w, "Failed to save to database", http.StatusInternalServerError)
		return
	}

	image := models.Image{
		ID:        imageID,
		Filename:  filename,
		S3Key:     s3Key,
		S3URL:     s3URL,
		Category:  category,
		CreatedAt: time.Now(),
		IsActive:  true,
	}

	response := models.UploadResponse{
		Message: "Image uploaded successfully",
		Image:   &image,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (h *ImageHandler) DeleteImage(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid image ID", http.StatusBadRequest)
		return
	}

	var img models.Image
	err = h.db.QueryRow(`
		SELECT id, filename, s3_key, s3_url, category, created_at, is_active 
		FROM images 
		WHERE id = $1`,
		id,
	).Scan(&img.ID, &img.Filename, &img.S3Key, &img.S3URL, &img.Category, &img.CreatedAt, &img.IsActive)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Image not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	bucketName := os.Getenv("S3_BUCKET_NAME")
	_, err = h.s3Client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(img.S3Key),
	})
	if err != nil {
		http.Error(w, "Failed to delete from S3", http.StatusInternalServerError)
		return
	}

	_, err = h.db.Exec("DELETE FROM images WHERE id = $1", id)
	if err != nil {
		http.Error(w, "Failed to delete from database", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Image deleted successfully",
	})
}

func isValidImageType(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	validExts := []string{".jpg", ".jpeg", ".png", ".webp"}
	
	for _, validExt := range validExts {
		if ext == validExt {
			return true
		}
	}
	return false
}

func getContentType(ext string) string {
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}