package handlers

import (
	"database/sql"
	"encoding/json"
	"go-postgres-example/pkg/config"
	"go-postgres-example/pkg/db"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// SongHandler holds the dependencies for the song handlers.
type SongHandler struct {
	DB  *sql.DB
	Cfg *config.Config
}

// NewSongHandler creates a new SongHandler.
func NewSongHandler(db *sql.DB, cfg *config.Config) *SongHandler {
	return &SongHandler{DB: db, Cfg: cfg}
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
