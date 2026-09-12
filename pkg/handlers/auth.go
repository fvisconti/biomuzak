package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"go-postgres-example/pkg/auth"
	"go-postgres-example/pkg/config"
	"go-postgres-example/pkg/db"
	"go-postgres-example/pkg/middleware"
	"go-postgres-example/pkg/storage"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgconn"
)

// AuthHandler holds the dependencies for the auth handlers
type AuthHandler struct {
	DB      *sql.DB
	Cfg     *config.Config
	Storage storage.StorageService
}

// NewAuthHandler creates a new AuthHandler
func NewAuthHandler(db *sql.DB, cfg *config.Config, s storage.StorageService) *AuthHandler {
	return &AuthHandler{DB: db, Cfg: cfg, Storage: s}
}

// RegisterRequest represents the request body for user registration
type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

// Register handles user registration
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if !h.Cfg.AllowRegistration {
		SendError(w, "Registration is disabled", http.StatusForbidden)
		return
	}
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" || req.Email == "" {
		SendError(w, "Username, password and email are required", http.StatusBadRequest)
		return
	}

	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		SendError(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	// Check for existing username or email to return a clear error
	var existingID int
	err = h.DB.QueryRow("SELECT id FROM users WHERE username = $1 OR email = $2", req.Username, req.Email).Scan(&existingID)
	if err == nil {
		SendError(w, "Username or email already exists", http.StatusConflict)
		return
	} else if err != sql.ErrNoRows {
		SendError(w, "Failed checking existing users", http.StatusInternalServerError)
		return
	}

	_, err = h.DB.Exec("INSERT INTO users (username, email, password_hash) VALUES ($1, $2, $3)", req.Username, req.Email, hashedPassword)
	if err != nil {
		// A concurrent registration may have won the race between our
		// existence check and this insert. Surface it as a 409, not a 500.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			SendError(w, "Username or email already exists", http.StatusConflict)
			return
		}
		SendError(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	SendCreated(w, "User created successfully", nil)
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
		SendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		SendError(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	var userID int
	var passwordHash string
	err := h.DB.QueryRow("SELECT id, password_hash FROM users WHERE username = $1", req.Username).Scan(&userID, &passwordHash)
	if err != nil {
		if err == sql.ErrNoRows {
			SendError(w, "Invalid username or password", http.StatusUnauthorized)
			return
		}
		SendError(w, "Failed to query user", http.StatusInternalServerError)
		return
	}

	if !auth.CheckPasswordHash(req.Password, passwordHash) {
		SendError(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	token, err := auth.GenerateJWT(userID, h.Cfg.JWTSecret)
	if err != nil {
		SendError(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	SendData(w, map[string]string{"token": token})
}

// MeResponse represents the authenticated user information
type MeResponse struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	IsAdmin  bool   `json:"is_admin"`
}

// Me returns the current authenticated user's info
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		SendError(w, "Failed to get user ID from context", http.StatusInternalServerError)
		return
	}
	var resp MeResponse
	err := h.DB.QueryRow("SELECT id, username, COALESCE(is_admin, FALSE) FROM users WHERE id = $1", userID).Scan(&resp.ID, &resp.Username, &resp.IsAdmin)
	if err != nil {
		if err == sql.ErrNoRows {
			SendError(w, "User not found", http.StatusNotFound)
			return
		}
		SendError(w, "Failed to query user", http.StatusInternalServerError)
		return
	}
	SendData(w, resp)
}

// PublicConfigResponse exposes non-sensitive configuration to the frontend
type PublicConfigResponse struct {
	AllowRegistration bool `json:"allow_registration"`
}

// PublicConfig returns public configuration flags (no auth required)
func (h *AuthHandler) PublicConfig(w http.ResponseWriter, r *http.Request) {
	SendData(w, PublicConfigResponse{AllowRegistration: h.Cfg.AllowRegistration})
}

// AdminCreateUserRequest represents the request body for admin-created users
type AdminCreateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email,omitempty"`
	IsAdmin  bool   `json:"is_admin,omitempty"`
}

// AdminCreateUser allows an admin to create a new user
func (h *AuthHandler) AdminCreateUser(w http.ResponseWriter, r *http.Request) {
	var req AdminCreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.Username == "" || req.Password == "" {
		SendError(w, "Username and password are required", http.StatusBadRequest)
		return
	}
	email := req.Email
	if email == "" {
		email = req.Username + "@local"
	}
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		SendError(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}
	// Only allow setting admin flag to true by admins; the route will already be protected by admin middleware
	isAdmin := req.IsAdmin
	_, err = h.DB.Exec("INSERT INTO users (username, email, password_hash, is_admin) VALUES ($1, $2, $3, $4)", req.Username, email, hashedPassword, isAdmin)
	if err != nil {
		SendError(w, "Failed to create user", http.StatusInternalServerError)
		return
	}
	SendCreated(w, "User created successfully", nil)
}

// AdminListUsers returns a list of all users (admin-only)
func (h *AuthHandler) AdminListUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query("SELECT id, username, email, COALESCE(is_admin, FALSE) FROM users ORDER BY id")
	if err != nil {
		SendError(w, "Failed to query users", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type userRow struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
		IsAdmin  bool   `json:"is_admin"`
	}

	var users []userRow
	for rows.Next() {
		var u userRow
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.IsAdmin); err != nil {
			SendError(w, "Failed scanning users", http.StatusInternalServerError)
			return
		}
		users = append(users, u)
	}

	SendData(w, users)
}

// DeleteUser allows an admin to delete a user by ID
func (h *AuthHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		SendError(w, "Missing user id", http.StatusBadRequest)
		return
	}
	var userID int
	_, err := fmt.Sscanf(idStr, "%d", &userID)
	if err != nil {
		SendError(w, "Invalid user id", http.StatusBadRequest)
		return
	}

	// Find songs that only this user references BEFORE deleting the user, so
	// we can garbage-collect them (DB rows + storage objects) afterward.
	orphaned, err := db.GetOrphanedSongsForUser(h.DB, userID)
	if err != nil {
		log.Printf("Warning: failed to find orphaned songs for user %d: %v", userID, err)
	}

	res, err := h.DB.Exec("DELETE FROM users WHERE id = $1", userID)
	if err != nil {
		SendError(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		SendError(w, "User not found", http.StatusNotFound)
		return
	}

	// Garbage-collect songs that no other user references.
	if len(orphaned) > 0 {
		ids := make([]int, len(orphaned))
		for i, s := range orphaned {
			ids[i] = s.ID
		}
		if err := db.DeleteSongs(h.DB, ids); err != nil {
			log.Printf("Warning: failed to delete orphaned songs for user %d: %v", userID, err)
		}
		if h.Storage != nil {
			for _, s := range orphaned {
				if err := h.Storage.DeleteObject(r.Context(), s.FilePath); err != nil {
					log.Printf("Warning: failed to delete storage object %s: %v", s.FilePath, err)
				}
			}
		}
	}

	w.WriteHeader(http.StatusNoContent)
}
