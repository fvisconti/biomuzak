package handlers

import (
	"database/sql"
	"encoding/json"
	"go-postgres-example/pkg/config"
	"go-postgres-example/pkg/db"
	"go-postgres-example/pkg/storage"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// SongHandler holds the dependencies for the song handlers.
type SongHandler struct {
	DB      *sql.DB
	Cfg     *config.Config
	Storage storage.StorageService
}

// NewSongHandler creates a new SongHandler.
func NewSongHandler(db *sql.DB, cfg *config.Config, s storage.StorageService) *SongHandler {
	return &SongHandler{DB: db, Cfg: cfg, Storage: s}
}

// GetSimilarSongsHandler handles finding songs similar to a given song.
func (h *SongHandler) GetSimilarSongsHandler(w http.ResponseWriter, r *http.Request) {
	songIDStr := chi.URLParam(r, "songID")
	songID, err := strconv.Atoi(songIDStr)
	if err != nil {
		http.Error(w, "Invalid song ID", http.StatusBadRequest)
		return
	}

	// 1. Get the embedding for the query song.
	queryEmbedding, err := db.GetSongEmbedding(h.DB, songID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Embedding not found for the given song", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to get song embedding", http.StatusInternalServerError)
		}
		return
	}

	// 2. Find similar songs in the database.
	// We'll fetch the top 5 similar songs, excluding the query song itself.
	similarSongs, err := db.FindSimilarSongs(h.DB, songID, queryEmbedding, 5)
	if err != nil {
		http.Error(w, "Failed to find similar songs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(similarSongs)
}

// DeleteSongHandler permanently deletes a song from the catalog (DB row +
// storage object). This is an admin-only operation: the catalog is shared,
// so deleting a song removes it for all users. Cascading FKs remove the
// song's user_songs ratings, embeddings, and playlist entries.
func (h *SongHandler) DeleteSongHandler(w http.ResponseWriter, r *http.Request) {
	songIDStr := chi.URLParam(r, "songID")
	songID, err := strconv.Atoi(songIDStr)
	if err != nil {
		http.Error(w, "Invalid song ID", http.StatusBadRequest)
		return
	}

	// Fetch the file path before deleting the row.
	filePath, err := db.GetSongFilePath(h.DB, songID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Song not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to look up song", http.StatusInternalServerError)
		return
	}

	// Delete the DB row and the storage object concurrently — they are
	// independent (the file path is already resolved).
	type delResult struct {
		err error
	}
	dbCh := make(chan delResult, 1)
	go func() {
		_, err := h.DB.Exec("DELETE FROM songs WHERE id = $1", songID)
		dbCh <- delResult{err: err}
	}()

	var storageErr error
	if h.Storage != nil {
		storageErr = h.Storage.DeleteObject(r.Context(), filePath)
	}

	dbRes := <-dbCh
	if dbRes.err != nil {
		http.Error(w, "Failed to delete song", http.StatusInternalServerError)
		return
	}
	if storageErr != nil {
		// DB row is gone; storage cleanup is best-effort.
		http.Error(w, "Song deleted from catalog but storage cleanup failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
