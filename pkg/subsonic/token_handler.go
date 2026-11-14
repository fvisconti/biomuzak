package subsonic

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"go-postgres-example/pkg/config"
	"go-postgres-example/pkg/middleware"
)

// TokenHandler exposes the derived Subsonic API token of the authenticated
// user so it can be pasted into third-party Subsonic clients.
type TokenHandler struct {
	DB  *sql.DB
	Cfg *config.Config
}

// NewTokenHandler creates a TokenHandler.
func NewTokenHandler(db *sql.DB, cfg *config.Config) *TokenHandler {
	return &TokenHandler{DB: db, Cfg: cfg}
}

// GetSubsonicTokenHandler handles GET /api/subsonic/token (JWT-protected in
// the main router). The token is derived from the verified user claim.
func (h *TokenHandler) GetSubsonicTokenHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var username string
	if err := h.DB.QueryRowContext(r.Context(),
		"SELECT username FROM users WHERE id = $1", userID).Scan(&username); err != nil {
		http.Error(w, "Failed to look up user", http.StatusInternalServerError)
		return
	}

	token := DeriveSubsonicToken(h.Cfg.JWTSecret, username)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}
