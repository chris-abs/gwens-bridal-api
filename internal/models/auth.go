package models

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type User struct {
	Username string `json:"username"`
	Role     string `json:"role"`
}

type UploadResponse struct {
	Message string `json:"message"`
	Image   *Image `json:"image,omitempty"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}