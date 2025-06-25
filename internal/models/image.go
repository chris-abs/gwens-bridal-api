package models

import "time"

type Image struct {
	ID        int       `json:"id" db:"id"`
	Filename  string    `json:"filename" db:"filename"`
	S3Key     string    `json:"s3_key" db:"s3_key"`
	S3URL     string    `json:"s3_url" db:"s3_url"`
	Category  string    `json:"category" db:"category"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	IsActive  bool      `json:"is_active" db:"is_active"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

type UploadResponse struct {
	Message string `json:"message"`
	Image   *Image `json:"image,omitempty"`
}