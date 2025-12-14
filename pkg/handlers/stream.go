package handlers

import (
	"database/sql"
	"go-postgres-example/pkg/config"
	"go-postgres-example/pkg/db"
	"go-postgres-example/pkg/middleware"
	"go-postgres-example/pkg/storage"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

// StreamHandler holds dependencies for streaming
type StreamHandler struct {
	DB      *sql.DB
	Cfg     *config.Config
	Storage storage.StorageService
}

// NewStreamHandler creates a new StreamHandler
func NewStreamHandler(db *sql.DB, cfg *config.Config, s storage.StorageService) *StreamHandler {
	return &StreamHandler{DB: db, Cfg: cfg, Storage: s}
}

// StreamSongHandler streams a song's audio content
func (h *StreamHandler) StreamSongHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Auth check (optional, but good practice. Maybe we want public links later? For now, strict.)
	_, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	songIDStr := chi.URLParam(r, "songID")
	songID, err := strconv.Atoi(songIDStr)
	if err != nil {
		http.Error(w, "Invalid song ID", http.StatusBadRequest)
		return
	}

	// 2. Get file path from DB
	filePath, err := db.GetSongFilePath(h.DB, songID)
	if err != nil {
		http.Error(w, "Song not found", http.StatusNotFound)
		return
	}

	// 3. Get file stream from Storage
	fileStream, err := h.Storage.GetFileStream(r.Context(), filePath)
	if err != nil {
		http.Error(w, "Failed to retrieve song file", http.StatusInternalServerError)
		return
	}
	defer fileStream.Close()

	// 4. Serve the content
	// We should try to set content type if known.
	// For now, let ServeContent handle ranges if we pass a Seeker.
	// storage.ReadSeekCloser implements Read, Seek, Close.

	// Determine content type from extension (naive but effective)
	// Or we could store logic.

	// http.ServeContent efficiently handles Range requests which is crucial for seeking.
	http.ServeContent(w, r, "audio", time.Time{}, fileStream)
}
