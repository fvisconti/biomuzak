package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"go-postgres-example/pkg/config"
	"go-postgres-example/pkg/middleware"
	"go-postgres-example/pkg/storage"
	"go-postgres-example/pkg/tidal"

	"github.com/go-chi/chi/v5"
)

// TidalHandler handles Tidal pairing, browsing, and import.
type TidalHandler struct {
	DB       *sql.DB
	Cfg      *config.Config
	Storage  storage.StorageService
	Auth     *tidal.AuthClient
	Client   *tidal.Client
	Importer *tidal.Importer

	// pendingPairs holds in-flight device authorizations.
	pendingMu    sync.Mutex
	pendingPairs map[string]*pendingPair
}

type pendingPair struct {
	deviceCode string
	userID     int
	expiresAt  time.Time
}

// NewTidalHandler builds a TidalHandler.
func NewTidalHandler(db *sql.DB, cfg *config.Config, s storage.StorageService) *TidalHandler {
	auth := tidal.NewAuthClient(cfg.TidalClientID, cfg.TidalClientSecret, db)
	client := tidal.NewClient(auth)
	importer := tidal.NewImporter(client, db, s)
	return &TidalHandler{
		DB:           db,
		Cfg:          cfg,
		Storage:      s,
		Auth:         auth,
		Client:       client,
		Importer:     importer,
		pendingPairs: make(map[string]*pendingPair),
	}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// ---- Pairing ----

// PairStartHandler initiates the device authorization flow.
func (h *TidalHandler) PairStartHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if h.Cfg.TidalClientID == "" || h.Cfg.TidalClientSecret == "" {
		http.Error(w, "Tidal is not configured on the server", http.StatusServiceUnavailable)
		return
	}

	dar, err := h.Auth.StartDeviceAuth(r.Context())
	if err != nil {
		http.Error(w, "Failed to start Tidal pairing", http.StatusBadGateway)
		return
	}

	pairID := fmt.Sprintf("pair-%d-%d", time.Now().UnixNano(), userID)
	h.pendingMu.Lock()
	h.pendingPairs[pairID] = &pendingPair{
		deviceCode: dar.DeviceCode,
		userID:     userID,
		expiresAt:  time.Now().Add(time.Duration(dar.ExpiresIn) * time.Second),
	}
	h.pendingMu.Unlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"pairId":                  pairID,
		"userCode":                dar.UserCode,
		"verificationUri":         dar.VerificationURI,
		"verificationUriComplete": dar.VerificationURIComplete,
		"expiresIn":               dar.ExpiresIn,
		"interval":                dar.Interval,
	})
}

// PairStatusHandler polls the device authorization status.
func (h *TidalHandler) PairStatusHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	pairID := r.URL.Query().Get("pairId")

	h.pendingMu.Lock()
	pair, exists := h.pendingPairs[pairID]
	h.pendingMu.Unlock()

	if !exists {
		writeJSON(w, http.StatusOK, map[string]string{"status": "expired"})
		return
	}
	if pair.userID != userID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	if time.Now().After(pair.expiresAt) {
		h.removePair(pairID)
		writeJSON(w, http.StatusOK, map[string]string{"status": "expired"})
		return
	}

	tr, err := h.Auth.PollDeviceAuth(r.Context(), pair.deviceCode)
	switch {
	case errors.Is(err, tidal.ErrAuthorizationPending):
		writeJSON(w, http.StatusOK, map[string]string{"status": "pending"})
		return
	case errors.Is(err, tidal.ErrSlowDown):
		writeJSON(w, http.StatusOK, map[string]string{"status": "pending"})
		return
	case errors.Is(err, tidal.ErrAuthorizationExpired):
		h.removePair(pairID)
		writeJSON(w, http.StatusOK, map[string]string{"status": "expired"})
		return
	case err != nil:
		http.Error(w, "Failed to poll Tidal pairing", http.StatusBadGateway)
		return
	}

	// Success: persist credentials.
	creds := &tidal.Credentials{
		UserID:               userID,
		TidalUserID:          tr.User.UserID,
		TidalUsername:        tr.User.Username,
		AccessToken:          tr.AccessToken,
		RefreshToken:         tr.RefreshToken,
		AccessTokenExpiresAt: time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second),
		Config:               tidal.DefaultConfig(),
	}
	if err := tidal.UpsertCredentials(r.Context(), h.DB, creds); err != nil {
		http.Error(w, "Failed to save Tidal connection", http.StatusInternalServerError)
		return
	}
	h.removePair(pairID)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":        "authorized",
		"tidalUsername": tr.User.Username,
	})
}

func (h *TidalHandler) removePair(pairID string) {
	h.pendingMu.Lock()
	defer h.pendingMu.Unlock()
	delete(h.pendingPairs, pairID)
}

// ---- Status / settings / disconnect ----

// StatusHandler returns the user's Tidal connection status + config.
func (h *TidalHandler) StatusHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	creds, err := tidal.GetCredentials(r.Context(), h.DB, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			cfg, _ := tidal.GetConfig(r.Context(), h.DB, userID)
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"connected": false,
				"config":    cfg,
			})
			return
		}
		http.Error(w, "Failed to load Tidal status", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"connected":     true,
		"tidalUsername": creds.TidalUsername,
		"expiresAt":     creds.AccessTokenExpiresAt,
		"config":        creds.Config,
	})
}

// SettingsHandler persists the user's Tidal config.
func (h *TidalHandler) SettingsHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var cfg tidal.TidalConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	validated, err := cfg.Validate()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Ensure a credentials row exists so we can attach the config.
	if _, err := tidal.GetCredentials(r.Context(), h.DB, userID); err == sql.ErrNoRows {
		// No connection yet: create a placeholder row so config persists.
		placeholder := &tidal.Credentials{
			UserID:               userID,
			AccessToken:          "",
			RefreshToken:         "",
			AccessTokenExpiresAt: time.Now(),
			Config:               validated,
		}
		if err := tidal.UpsertCredentials(r.Context(), h.DB, placeholder); err != nil {
			http.Error(w, "Failed to save settings", http.StatusInternalServerError)
			return
		}
	} else if err != nil {
		http.Error(w, "Failed to save settings", http.StatusInternalServerError)
		return
	}

	if err := tidal.SaveConfig(r.Context(), h.DB, userID, validated); err != nil {
		http.Error(w, "Failed to save settings", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"config": validated})
}

// DisconnectHandler removes the user's Tidal connection.
func (h *TidalHandler) DisconnectHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if err := tidal.DeleteCredentials(r.Context(), h.DB, userID); err != nil {
		http.Error(w, "Failed to disconnect", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "disconnected"})
}

// ---- Browse ----

// LibraryHandler returns the user's Tidal library (tracks/albums/playlists).
func (h *TidalHandler) LibraryHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	kind := chi.URLParam(r, "kind")
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 50
	}

	var (
		page *tidal.Page
		err  error
	)
	switch kind {
	case "tracks":
		page, err = h.Client.GetLibraryTracks(r.Context(), userID, offset, limit)
	case "albums":
		page, err = h.Client.GetLibraryAlbums(r.Context(), userID, offset, limit)
	case "playlists":
		page, err = h.Client.GetLibraryPlaylists(r.Context(), userID, offset, limit)
	default:
		http.Error(w, "Invalid kind", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, "Failed to fetch Tidal library", http.StatusBadGateway)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

// AlbumHandler returns an album and its tracks.
func (h *TidalHandler) AlbumHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	albumID, err := strconv.Atoi(chi.URLParam(r, "albumID"))
	if err != nil {
		http.Error(w, "Invalid album ID", http.StatusBadRequest)
		return
	}
	album, err := h.Client.GetAlbum(r.Context(), userID, albumID)
	if err != nil {
		http.Error(w, "Failed to fetch album", http.StatusBadGateway)
		return
	}
	tracks, err := h.Client.GetAlbumTracks(r.Context(), userID, albumID, 0, 100)
	if err != nil {
		http.Error(w, "Failed to fetch album tracks", http.StatusBadGateway)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"album":  album,
		"tracks": tracks.Items,
	})
}

// PlaylistHandler returns a playlist and its tracks.
func (h *TidalHandler) PlaylistHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	playlistID, err := strconv.Atoi(chi.URLParam(r, "playlistID"))
	if err != nil {
		http.Error(w, "Invalid playlist ID", http.StatusBadRequest)
		return
	}
	pl, err := h.Client.GetPlaylist(r.Context(), userID, playlistID)
	if err != nil {
		http.Error(w, "Failed to fetch playlist", http.StatusBadGateway)
		return
	}
	tracks, err := h.Client.GetPlaylistTracks(r.Context(), userID, playlistID, 0, 200)
	if err != nil {
		http.Error(w, "Failed to fetch playlist tracks", http.StatusBadGateway)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"playlist": pl,
		"tracks":   tracks.Items,
	})
}

// ---- Import ----

// ImportRequest is the body for POST /api/tidal/import.
type ImportRequest struct {
	Tracks       []tidal.TrackMeta  `json:"tracks"`
	PlaylistName string             `json:"playlist_name"`
	Config       *tidal.TidalConfig `json:"config,omitempty"`
}

// startImportJob applies E3b source-genre matching and launches a detached
// batch import for the given tracks. Shared by the browse-based import and
// the link-paste import (E5).
func (h *TidalHandler) startImportJob(ctx context.Context, userID int, tracks []tidal.TrackMeta, playlistName string, reqCfg *tidal.TidalConfig) (string, error) {
	cfg, err := tidal.GetConfig(ctx, h.DB, userID)
	if err != nil {
		return "", fmt.Errorf("failed to load tidal config: %w", err)
	}
	if reqCfg != nil {
		if v, err := reqCfg.Validate(); err == nil {
			cfg = v
		}
	}

	jobID := tidal.NewJobID()

	// E3b: resolve the genre catalogue match for the source name once, before
	// the job starts, so every imported song inherits it. Non-fatal: the import
	// proceeds without an inferred genre when the match fails.
	if playlistName != "" {
		if _, err := h.Importer.ApplySourceGenre(ctx, tracks, playlistName); err != nil {
			log.Printf("genre match for import source %q failed: %v", playlistName, err)
		}
	}

	// importTrack closure: import + optionally add to playlist.
	importTrack := func(ctx context.Context, uid int, meta tidal.TrackMeta, c tidal.TidalConfig) (*tidal.ImportResult, error) {
		res, err := h.Importer.ImportTrack(ctx, uid, meta, c)
		if err != nil {
			return nil, err
		}
		if playlistName != "" {
			if err := h.Importer.AddToPlaylist(ctx, uid, res.SongID, playlistName); err != nil {
				// Non-fatal: track is imported even if playlist add fails.
				_ = err
			}
		}
		return res, nil
	}

	// Use a detached context so the job outlives the HTTP request.
	job := tidal.RunBatch(context.Background(), jobID, userID, tracks, cfg, importTrack)
	_ = job
	return jobID, nil
}

// ImportHandler starts a batch import job.
func (h *TidalHandler) ImportHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req ImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if len(req.Tracks) == 0 {
		http.Error(w, "No tracks to import", http.StatusBadRequest)
		return
	}

	jobID, err := h.startImportJob(r.Context(), userID, req.Tracks, req.PlaylistName, req.Config)
	if err != nil {
		log.Printf("start import job: %v", err)
		http.Error(w, "Failed to start import", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"jobId": jobID})
}

// LinkImportRequest is the body for POST /api/tidal/import/link (E5).
type LinkImportRequest struct {
	URL string `json:"url"`
}

// resolveLink maps a parsed Tidal link to importable tracks plus the source
// name (album/playlist title) used for E3b genre matching.
func (h *TidalHandler) resolveLink(ctx context.Context, userID int, link *tidal.TidalLink) ([]tidal.TrackMeta, string, error) {
	switch link.Type {
	case tidal.LinkTrack:
		id, err := strconv.Atoi(link.ID)
		if err != nil {
			return nil, "", fmt.Errorf("invalid track id: %w", err)
		}
		res, err := h.Client.GetTrack(ctx, userID, id)
		if err != nil {
			return nil, "", err
		}
		return resourcesToMetas([]tidal.Resource{*res}), "", nil
	case tidal.LinkAlbum:
		id, err := strconv.Atoi(link.ID)
		if err != nil {
			return nil, "", fmt.Errorf("invalid album id: %w", err)
		}
		album, err := h.Client.GetAlbum(ctx, userID, id)
		if err != nil {
			return nil, "", err
		}
		items, err := h.fetchAllPage(func(offset, limit int) (*tidal.Page, error) {
			return h.Client.GetAlbumTracks(ctx, userID, id, offset, limit)
		})
		if err != nil {
			return nil, "", err
		}
		return resourcesToMetas(items), album.Title, nil
	case tidal.LinkPlaylist:
		pl, err := h.Client.GetPlaylistByURLID(ctx, userID, link.ID)
		if err != nil {
			return nil, "", err
		}
		items, err := h.fetchAllPage(func(offset, limit int) (*tidal.Page, error) {
			return h.Client.GetPlaylistTracksByURLID(ctx, userID, link.ID, offset, limit)
		})
		if err != nil {
			return nil, "", err
		}
		return resourcesToMetas(items), pl.Title, nil
	}
	return nil, "", fmt.Errorf("unsupported link type")
}

// fetchAllPage pages through a Tidal list endpoint, capped so a pathological
// totalNumItems cannot loop the import forever.
func (h *TidalHandler) fetchAllPage(page func(offset, limit int) (*tidal.Page, error)) ([]tidal.Resource, error) {
	const pageSize = 500
	const maxTotal = 2000

	var all []tidal.Resource
	offset := 0
	for len(all) < maxTotal {
		p, err := page(offset, pageSize)
		if err != nil {
			return nil, err
		}
		all = append(all, p.Items...)
		offset += len(p.Items)
		if len(p.Items) == 0 || offset >= p.TotalNumItems {
			break
		}
	}
	return all, nil
}

// resourcesToMetas converts Tidal resources to import metadata, skipping
// video items (not music).
func resourcesToMetas(items []tidal.Resource) []tidal.TrackMeta {
	out := make([]tidal.TrackMeta, 0, len(items))
	for _, it := range items {
		if it.IsVideo {
			continue
		}
		out = append(out, tidal.TrackMeta{
			TrackID: it.ID,
			Title:   it.Title,
			Artist:  it.ArtistName,
			Album:   it.AlbumName,
			Year:    it.Year,
		})
	}
	return out
}

// LinkImportHandler resolves a pasted Tidal track/album/playlist URL and
// starts the same batch import as the browse flow (E5).
func (h *TidalHandler) LinkImportHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req LinkImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	link, err := tidal.ParseTidalURL(req.URL)
	if err != nil {
		http.Error(w, "Not a recognized Tidal link (track, album, or playlist)", http.StatusBadRequest)
		return
	}

	tracks, playlistName, err := h.resolveLink(r.Context(), userID, link)
	if err != nil {
		if errors.Is(err, tidal.ErrNotConnected) {
			http.Error(w, "Not connected to Tidal", http.StatusServiceUnavailable)
			return
		}
		log.Printf("tidal link resolve failed: %v", err)
		http.Error(w, "Failed to resolve Tidal link", http.StatusBadGateway)
		return
	}
	if len(tracks) == 0 {
		http.Error(w, "No importable tracks found in link", http.StatusNotFound)
		return
	}

	jobID, err := h.startImportJob(r.Context(), userID, tracks, playlistName, nil)
	if err != nil {
		log.Printf("start import job: %v", err)
		http.Error(w, "Failed to start import", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"jobId": jobID})
}

// ImportStatusHandler returns the status of an import job.
func (h *TidalHandler) ImportStatusHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	jobID := r.URL.Query().Get("jobId")
	job, exists := tidal.GetImportJob(jobID)
	if !exists {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}
	if job.UserID != userID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	writeJSON(w, http.StatusOK, job.Snapshot())
}
