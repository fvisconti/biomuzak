package handlers

import (
	"database/sql"
	"encoding/json"
	"go-postgres-example/pkg/config"
	"go-postgres-example/pkg/db"
	"go-postgres-example/pkg/middleware"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// LibraryHandler holds the dependencies for the library handlers
type LibraryHandler struct {
	DB  *sql.DB
	Cfg *config.Config
}

// NewLibraryHandler creates a new LibraryHandler
func NewLibraryHandler(db *sql.DB, cfg *config.Config) *LibraryHandler {
	return &LibraryHandler{DB: db, Cfg: cfg}
}

// GetLibraryHandler handles fetching a user's library
func (h *LibraryHandler) GetLibraryHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Failed to get user ID from context", http.StatusInternalServerError)
		return
	}

	// Parse query parameters
	queryParams := r.URL.Query()
	filters := map[string]string{
		"genre":  queryParams.Get("genre"),
		"artist": queryParams.Get("artist"),
		"album":  queryParams.Get("album"),
		"year":   queryParams.Get("year"),
	}
	sortBy := queryParams.Get("sort_by")

	songs, err := db.GetSongsByUserID(h.DB, userID, filters, sortBy)
	if err != nil {
		http.Error(w, "Failed to get songs from database", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(songs)
}

// RateSongRequest represents the request body for rating a song
type RateSongRequest struct {
	Rating int `json:"rating"`
}

// RateSongHandler handles rating a song
func (h *LibraryHandler) RateSongHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Failed to get user ID from context", http.StatusInternalServerError)
		return
	}

	songIDStr := chi.URLParam(r, "songID")
	songID, err := strconv.Atoi(songIDStr)
	if err != nil {
		http.Error(w, "Invalid song ID", http.StatusBadRequest)
		return
	}

	var req RateSongRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Rating < 1 || req.Rating > 5 {
		http.Error(w, "Rating must be between 1 and 5", http.StatusBadRequest)
		return
	}

	if err := db.RateSong(h.DB, userID, songID, req.Rating); err != nil {
		http.Error(w, "Failed to rate song", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Song rated successfully"})
}
