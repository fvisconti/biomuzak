package handlers

import (
	"database/sql"
	"encoding/json"
	"go-postgres-example/pkg/auth"
	"go-postgres-example/pkg/config"
	"net/http"
)

// AuthHandler holds the dependencies for the auth handlers
type AuthHandler struct {
	DB  *sql.DB
	Cfg *config.Config
}

// NewAuthHandler creates a new AuthHandler
func NewAuthHandler(db *sql.DB, cfg *config.Config) *AuthHandler {
	return &AuthHandler{DB: db, Cfg: cfg}
}

// RegisterRequest represents the request body for user registration
type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

// Register handles user registration
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" || req.Email == "" {
		http.Error(w, "Username, password and email are required", http.StatusBadRequest)
		return
	}

	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	_, err = h.DB.Exec("INSERT INTO users (username, email, password_hash) VALUES ($1, $2, $3)", req.Username, req.Email, hashedPassword)
	if err != nil {
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "User created successfully"})
}

// LoginRequest represents the request body for user login
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Login handles user login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	var userID int
	var passwordHash string
	err := h.DB.QueryRow("SELECT id, password_hash FROM users WHERE username = $1", req.Username).Scan(&userID, &passwordHash)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)
			return
		}
		http.Error(w, "Failed to query user", http.StatusInternalServerError)
		return
	}

	if !auth.CheckPasswordHash(req.Password, passwordHash) {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	token, err := auth.GenerateJWT(userID, h.Cfg.JWTSecret)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}
