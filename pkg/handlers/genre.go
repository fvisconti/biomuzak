package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"go-postgres-example/pkg/db"
)

// GenreHandler manages the genre catalogue. Routes are admin-gated by the
// router (middleware.AdminOnly group), matching the other admin endpoints.
type GenreHandler struct {
	DB *sql.DB
}

// NewGenreHandler builds a GenreHandler.
func NewGenreHandler(db *sql.DB) *GenreHandler {
	return &GenreHandler{DB: db}
}

// ListHandler handles GET /api/admin/genres.
func (h *GenreHandler) ListHandler(w http.ResponseWriter, r *http.Request) {
	genres, err := db.ListGenres(h.DB)
	if err != nil {
		http.Error(w, "Failed to list genres", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(genres)
}

// CreateHandler handles POST /api/admin/genres with body {"name": "..."}.
func (h *GenreHandler) CreateHandler(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Name) == "" {
		http.Error(w, "Genre name is required", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(body.Name)
	if len([]rune(name)) > 255 {
		http.Error(w, "Genre name too long (max 255 characters)", http.StatusBadRequest)
		return
	}

	g, err := db.CreateGenre(h.DB, name)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "Genre already exists", http.StatusConflict)
		return
	}
	if err != nil {
		http.Error(w, "Failed to create genre", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(g)
}

// DeleteHandler handles DELETE /api/admin/genres/{genreID}. Songs keep their
// data; the FK ON DELETE SET NULL clears their genre.
func (h *GenreHandler) DeleteHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "genreID"))
	if err != nil {
		http.Error(w, "Invalid genre ID", http.StatusBadRequest)
		return
	}
	if err := db.DeleteGenre(h.DB, id); errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "Genre not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Failed to delete genre", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
