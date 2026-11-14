package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"go-postgres-example/pkg/config"
	"go-postgres-example/pkg/db"
	"go-postgres-example/pkg/middleware"
	"go-postgres-example/pkg/storage"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// LibraryHandler holds the dependencies for the library handlers
type LibraryHandler struct {
	DB  *sql.DB
	Cfg *config.Config
	// Storage is used to keep object tags in sync on genre changes (E4);
	// nil is tolerated (no tagging).
	Storage storage.StorageService
}

// NewLibraryHandler creates a new LibraryHandler
func NewLibraryHandler(db *sql.DB, cfg *config.Config, storage storage.StorageService) *LibraryHandler {
	return &LibraryHandler{DB: db, Cfg: cfg, Storage: storage}
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
		"q":      queryParams.Get("q"),
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

// DeleteSongHandler removes a song from the user's library
func (h *LibraryHandler) DeleteSongHandler(w http.ResponseWriter, r *http.Request) {
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

	if err := db.DeleteUserSong(h.DB, userID, songID); err != nil {
		http.Error(w, "Failed to delete song", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Song deleted successfully"})
}

// UpdateGenreRequest represents the request body for updating a song's genre
type UpdateGenreRequest struct {
	Genre string `json:"genre"`
}

// UpdateSongGenreHandler updates the genre of a song. The catalog is shared,
// so any admin may update any song's genre (route is behind AdminOnly).
func (h *LibraryHandler) UpdateSongGenreHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := middleware.GetUserIDFromContext(r.Context()); !ok {
		http.Error(w, "Failed to get user ID from context", http.StatusInternalServerError)
		return
	}

	songIDStr := chi.URLParam(r, "songID")
	songID, err := strconv.Atoi(songIDStr)
	if err != nil {
		http.Error(w, "Invalid song ID", http.StatusBadRequest)
		return
	}

	var req UpdateGenreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := db.UpdateSongGenre(h.DB, songID, req.Genre); err != nil {
		http.Error(w, "Failed to update genre", http.StatusInternalServerError)
		return
	}

	// E4: keep the stored object's genre tag in sync (best-effort — the
	// genre change itself already succeeded).
	h.retagSongObject(r.Context(), songID, req.Genre)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Genre updated successfully"})
}

// retagSongObject refreshes the genre tag on the song's stored object,
// preserving any other existing tags (e.g. artist). All failures are logged,
// never returned: tagging is informational metadata.
func (h *LibraryHandler) retagSongObject(ctx context.Context, songID int, genreName string) {
	if h.Storage == nil {
		return
	}
	filePath, err := db.GetSongFilePath(ctx, h.DB, songID)
	if err != nil {
		log.Printf("failed to get file path for song %d: %v", songID, err)
		return
	}
	tags, err := h.Storage.GetObjectTags(ctx, filePath)
	if err != nil {
		log.Printf("failed to read tags for %s: %v", filePath, err)
		return
	}
	if tags == nil {
		tags = map[string]string{}
	}
	tags["genre"] = genreName
	if err := h.Storage.TagObject(ctx, filePath, tags); err != nil {
		log.Printf("failed to retag object %s: %v", filePath, err)
	}
}

// SearchHandler handles fetching search results for artists, albums, and songs
func (h *LibraryHandler) SearchHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"artists": []interface{}{},
			"albums":  []interface{}{},
			"songs":   []interface{}{},
		})
		return
	}

	artists, albums, songs, err := db.Search(h.DB, query)
	if err != nil {
		log.Printf("Search failed: %v", err)
		http.Error(w, "Search failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"artists": artists,
		"albums":  albums,
		"songs":   songs,
	})
}
