package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go-postgres-example/pkg/config"
	"go-postgres-example/pkg/export"
	"go-postgres-example/pkg/middleware"
	"go-postgres-example/pkg/storage"
	"go-postgres-example/pkg/tidal"
)

// ExportHandler handles exporting library songs to a local folder.
type ExportHandler struct {
	DB      *sql.DB
	Cfg     *config.Config
	Storage storage.StorageService
	Exporter *export.Exporter
}

// NewExportHandler builds an ExportHandler.
func NewExportHandler(db *sql.DB, cfg *config.Config, s storage.StorageService) *ExportHandler {
	return &ExportHandler{
		DB:       db,
		Cfg:      cfg,
		Storage:  s,
		Exporter: export.NewExporter(db, s),
	}
}

// ExportRequest is the body for POST /api/export.
type ExportRequest struct {
	SongIDs []int `json:"song_ids"`
}

// ExportHandler_Start starts an export job.
func (h *ExportHandler) StartHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req ExportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if len(req.SongIDs) == 0 {
		http.Error(w, "No songs to export", http.StatusBadRequest)
		return
	}

	cfg, err := tidal.GetConfig(r.Context(), h.DB, userID)
	if err != nil {
		http.Error(w, "Failed to load export config", http.StatusInternalServerError)
		return
	}
	if cfg.ExportPath == "" {
		http.Error(w, "Set an export path in Tidal settings first", http.StatusBadRequest)
		return
	}

	jobID := fmt.Sprintf("export-%d", time.Now().UnixNano())
	job, err := h.Exporter.ExportSongs(r.Context(), jobID, userID, req.SongIDs, cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_ = job
	writeJSON(w, http.StatusAccepted, map[string]string{"jobId": jobID})
}

// ExportStatusHandler returns the status of an export job.
func (h *ExportHandler) StatusHandler(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserIDFromContext(r.Context())
	_ = userID
	jobID := r.URL.Query().Get("jobId")
	job, exists := export.GetExportJob(jobID)
	if !exists {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, job.Snapshot())
}
