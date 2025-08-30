package handlers

import (
	"database/sql"
	"encoding/json"
	"go-postgres-example/pkg/config"
	"go-postgres-example/pkg/db"
	"go-postgres-example/pkg/middleware"
	"go-postgres-example/pkg/models"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// PlaylistHandler holds the dependencies for the playlist handlers.
type PlaylistHandler struct {
	DB  *sql.DB
	Cfg *config.Config
}

// NewPlaylistHandler creates a new PlaylistHandler.
func NewPlaylistHandler(db *sql.DB, cfg *config.Config) *PlaylistHandler {
	return &PlaylistHandler{DB: db, Cfg: cfg}
}

// CreatePlaylistRequest defines the structure for the create playlist request.
type CreatePlaylistRequest struct {
	Name string `json:"name"`
}

// CreatePlaylistHandler handles the creation of a new playlist.
func (h *PlaylistHandler) CreatePlaylistHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Failed to get user ID from context", http.StatusInternalServerError)
		return
	}

	var req CreatePlaylistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Playlist name cannot be empty", http.StatusBadRequest)
		return
	}

	playlist, err := db.CreatePlaylist(h.DB, userID, req.Name)
	if err != nil {
		http.Error(w, "Failed to create playlist", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(playlist)
}

// GetUserPlaylistsHandler handles fetching all playlists for the authenticated user.
func (h *PlaylistHandler) GetUserPlaylistsHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Failed to get user ID from context", http.StatusInternalServerError)
		return
	}

	playlists, err := db.GetUserPlaylists(h.DB, userID)
	if err != nil {
		http.Error(w, "Failed to get playlists", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(playlists)
}

// GetPlaylistHandler handles fetching a single playlist with its songs.
func (h *PlaylistHandler) GetPlaylistHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Failed to get user ID from context", http.StatusInternalServerError)
		return
	}

	playlistIDStr := chi.URLParam(r, "playlistID")
	playlistID, err := strconv.Atoi(playlistIDStr)
	if err != nil {
		http.Error(w, "Invalid playlist ID", http.StatusBadRequest)
		return
	}

	playlist, err := db.GetPlaylistByID(h.DB, userID, playlistID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Playlist not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to get playlist", http.StatusInternalServerError)
		}
		return
	}

	songs, err := db.GetPlaylistSongs(h.DB, playlistID)
	if err != nil {
		http.Error(w, "Failed to get playlist songs", http.StatusInternalServerError)
		return
	}

	fullPlaylist := models.FullPlaylist{
		ID:        playlist.ID,
		UserID:    playlist.UserID,
		Name:      playlist.Name,
		CreatedAt: playlist.CreatedAt,
		UpdatedAt: playlist.UpdatedAt,
		Songs:     songs,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fullPlaylist)
}

// UpdatePlaylistRequest defines the structure for the update playlist request.
type UpdatePlaylistRequest struct {
	Name string `json:"name"`
}

// UpdatePlaylistHandler handles updating a playlist's details (e.g., name).
func (h *PlaylistHandler) UpdatePlaylistHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Failed to get user ID from context", http.StatusInternalServerError)
		return
	}

	playlistIDStr := chi.URLParam(r, "playlistID")
	playlistID, err := strconv.Atoi(playlistIDStr)
	if err != nil {
		http.Error(w, "Invalid playlist ID", http.StatusBadRequest)
		return
	}

	var req UpdatePlaylistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Playlist name cannot be empty", http.StatusBadRequest)
		return
	}

	err = db.UpdatePlaylistName(h.DB, userID, playlistID, req.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Playlist not found or you don't have permission to update it", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to update playlist", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Playlist updated successfully"})
}

// DeletePlaylistHandler handles deleting a playlist.
func (h *PlaylistHandler) DeletePlaylistHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Failed to get user ID from context", http.StatusInternalServerError)
		return
	}

	playlistIDStr := chi.URLParam(r, "playlistID")
	playlistID, err := strconv.Atoi(playlistIDStr)
	if err != nil {
		http.Error(w, "Invalid playlist ID", http.StatusBadRequest)
		return
	}

	err = db.DeletePlaylist(h.DB, userID, playlistID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Playlist not found or you don't have permission to delete it", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to delete playlist", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Playlist deleted successfully"})
}

// AddSongToPlaylistRequest defines the structure for adding a song to a playlist.
type AddSongToPlaylistRequest struct {
	SongID   int `json:"song_id"`
	Position int `json:"position"` // Optional, defaults to end if 0 or less
}

// AddSongToPlaylistHandler handles adding a song to a playlist.
func (h *PlaylistHandler) AddSongToPlaylistHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Failed to get user ID from context", http.StatusInternalServerError)
		return
	}

	playlistIDStr := chi.URLParam(r, "playlistID")
	playlistID, err := strconv.Atoi(playlistIDStr)
	if err != nil {
		http.Error(w, "Invalid playlist ID", http.StatusBadRequest)
		return
	}

	_, err = db.GetPlaylistByID(h.DB, userID, playlistID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Playlist not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to verify playlist", http.StatusInternalServerError)
		}
		return
	}

	var req AddSongToPlaylistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.SongID <= 0 {
		http.Error(w, "Invalid song ID", http.StatusBadRequest)
		return
	}

	err = db.AddSongToPlaylist(h.DB, playlistID, req.SongID, req.Position)
	if err != nil {
		http.Error(w, "Failed to add song to playlist", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Song added to playlist successfully"})
}

// RemoveSongFromPlaylistHandler handles removing a song from a playlist.
func (h *PlaylistHandler) RemoveSongFromPlaylistHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Failed to get user ID from context", http.StatusInternalServerError)
		return
	}

	playlistIDStr := chi.URLParam(r, "playlistID")
	playlistID, err := strconv.Atoi(playlistIDStr)
	if err != nil {
		http.Error(w, "Invalid playlist ID", http.StatusBadRequest)
		return
	}

	_, err = db.GetPlaylistByID(h.DB, userID, playlistID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Playlist not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to verify playlist", http.StatusInternalServerError)
		}
		return
	}

	songIDStr := chi.URLParam(r, "songID")
	songID, err := strconv.Atoi(songIDStr)
	if err != nil {
		http.Error(w, "Invalid song ID", http.StatusBadRequest)
		return
	}

	err = db.RemoveSongFromPlaylist(h.DB, playlistID, songID)
	if err != nil {
		http.Error(w, "Failed to remove song from playlist", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Song removed from playlist successfully"})
}
